package main

import (
	"archive/tar"
	"bufio"
	"encoding/binary"
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/dustin/go-humanize"
)

type discoverCommand struct {
	dumpCommand
}

func (c *discoverCommand) Execute(args []string) error {
	processDump(c.Stdin, c.URL, c.Output, discover)
	return nil
}

func discover(r *tar.Reader, w io.Writer) int64 {
	const numTasks = 2
	var (
		processed int64
		status    int
		i         int
		stars     map[uint32]uint32
		reposPath string
	)

	for header, err := r.Next(); err != io.EOF; header, err = r.Next() {
		if err != nil {
			fail("reading tar.gz", err)
		}

		i++
		processed += header.Size
		isWatchers := strings.HasSuffix(header.Name, "watchers.csv")
		isProjects := strings.HasSuffix(header.Name, "projects.csv")
		mark := " "
		if isWatchers || isProjects {
			mark = ">"
		}

		strSize := humanize.Bytes(uint64(header.Size))
		if strings.HasSuffix(strSize, " B") {
			strSize += " "
		}

		if i == 1 {
			fmt.Print("\r", strings.Repeat(" ", 80))
		}

		fmt.Printf("\r%s %2d  %7s  %s\n", mark, i, strSize, header.Name)
		if isWatchers {
			stars = reduceWatchers(r)
			status++
		} else if isProjects {
			reposPath = reduceProjects(r)
			status++
		}

		if status == numTasks {
			break
		}
	}

	if stars != nil && reposPath != "" {
		writeData(w, stars, reposPath)
	}

	return processed
}

type repoTuple struct {
	name string
	num  uint32
}

func (t *repoTuple) encode(w io.Writer) error {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, uint32(len(t.name)))
	if _, err := w.Write(buf); err != nil {
		return err
	}

	if _, err := w.Write([]byte(t.name)); err != nil {
		return err
	}

	binary.LittleEndian.PutUint32(buf, t.num)
	if _, err := w.Write(buf); err != nil {
		return err
	}

	return nil
}

func (t *repoTuple) decode(r io.Reader) error {
	buf := make([]byte, 4)
	if _, err := r.Read(buf); err != nil {
		return err
	}

	size := binary.LittleEndian.Uint32(buf)
	buf = make([]byte, size)
	if _, err := r.Read(buf); err != nil {
		return err
	}

	t.name = string(buf)
	buf = make([]byte, 4)
	if _, err := r.Read(buf); err != nil {
		return err
	}

	t.num = binary.LittleEndian.Uint32(buf)
	return nil
}

type reposIter struct {
	r io.ReadCloser
}

func newReposIter(path string) (*reposIter, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	return &reposIter{f}, nil
}

func (i *reposIter) Next() (*repoTuple, error) {
	t := &repoTuple{}
	if err := t.decode(i.r); err != nil {
		return nil, err
	}

	return t, nil
}

func (i *reposIter) Close() error {
	return i.r.Close()
}

func reduceProjects(stream io.Reader) string {
	f, err := ioutil.TempFile("", "repos-id-")
	if err != nil {
		fail("reducing projects", err)
	}

	scanner := bufio.NewScanner(stream)
	skip := false
	tuple := &repoTuple{}
	for scanner.Scan() {
		line := scanner.Text()
		skipThis := skip
		skip = line[len(line)-1:] == "\\"
		if skipThis {
			continue
		}

		// Skip deleted repositories
		commaPos2 := strings.LastIndex(line, ",")
		commaPos2 = strings.LastIndex(line[:commaPos2], ",")
		commaPos1 := strings.LastIndex(line[:commaPos2], ",")
		deletedFlag := line[commaPos1+1 : commaPos2]
		if deletedFlag != "0" {
			continue
		}

		commaPos := strings.Index(line, ",")
		if commaPos < 0 {
			fail("parsing projects "+line, fmt.Errorf("comma not found"))
		}

		projectID, err := strconv.Atoi(line[:commaPos])
		if err != nil {
			fail(fmt.Sprintf("parsing projects project ID \"%s\"", line[:commaPos]), err)
		}

		if projectID < 0 {
			continue
		}

		line = line[commaPos+1+30:] // +"https://api.github.com/repos/
		commaPos = strings.Index(line, "\"")
		projectName := line[:commaPos]

		tuple.name = projectName
		tuple.num = uint32(projectID)
		if err := tuple.encode(f); err != nil {
			fail("reducing projects", err)
		}
	}

	if err := f.Close(); err != nil {
		fail("reducing projects", err)
	}

	reposPath, err := dedupRepos(f.Name())
	if err != nil {
		fail("reducing projects: deduplication", err)
	}

	if err := os.Remove(f.Name()); err != nil {
		fail("reducing projects", err)
	}

	return reposPath
}

func dedupRepos(path string) (string, error) {
	// note that a repository can be duplicated in the dump with
	// different ids because of an update:
	// 28974589,"https://api.github.com/repos/travisjeffery/ecs-deploy",4141,"ecs-update","Update ECS service to a Docker image.","Go","2015-11-19 08:18:23",\N,0,"2016-03-05 13:26:25",\N
	// 29621508,"https://api.github.com/repos/travisjeffery/ecs-deploy",4141,"ecs-deploy","Update ECS service to a Docker image.","Go","2015-11-19 08:18:23",\N,0,"2019-02-16 18:02:27",\N
	// the entry with greater id is the last update, since the
	// entries in the projects.csv file are sorted by id then the
	// last seen id for a certain project name is the correct one.

	w, err := ioutil.TempFile("", "dedup-repos-id-")
	if err != nil {
		return "", err
	}

	iter, err := newReposIter(path)
	if err != nil {
		return "", err
	}

	ids := map[string]uint32{}
	for tuple, err := iter.Next(); err != io.EOF; tuple, err = iter.Next() {
		if err != nil {
			return "", err
		}

		ids[tuple.name] = tuple.num
	}

	if err := iter.Close(); err != nil {
		return "", err
	}

	t := &repoTuple{}
	for t.name, t.num = range ids {
		if err := t.encode(w); err != nil {
			return "", err
		}
	}

	if err := w.Close(); err != nil {
		return "", nil
	}

	return w.Name(), nil
}

func reduceWatchers(stream io.Reader) map[uint32]uint32 {
	stars := map[uint32]uint32{}
	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		line := scanner.Text()
		commaPos := strings.Index(line, ",")
		projectID, err := strconv.Atoi(line[:commaPos])
		if err != nil {
			fail(fmt.Sprintf("parsing watchers project ID \"%s\"", line[:commaPos]), err)
		}

		stars[uint32(projectID)]++
	}

	return stars
}

func writeData(w io.Writer, stars map[uint32]uint32, reposPath string) {
	ri, err := newReposIter(reposPath)
	if err != nil {
		fail("writing to repositories file", err)
	}

	tupStars := make([]*repoTuple, 0, len(stars))
	noStars := []string{}
	for tupID, err := ri.Next(); err != io.EOF; tupID, err = ri.Next() {
		if err != nil {
			fail("writing to repositories file", err)
		}

		nstars, ok := stars[tupID.num]
		if !ok {
			noStars = append(noStars, tupID.name)
			continue
		}

		tupStars = append(tupStars, &repoTuple{
			name: tupID.name,
			num:  nstars,
		})
	}

	if err := ri.Close(); err != nil {
		fail("writing to repositories file", err)
	}

	if err := os.Remove(reposPath); err != nil {
		fail("writing to repositories file", err)
	}

	// Descending value order
	sort.Slice(tupStars, func(i, j int) bool {
		return tupStars[i].num > tupStars[j].num
	})

	cw := csv.NewWriter(w)
	headers := []string{"repository", "stars"}
	if err := cw.Write(headers); err != nil {
		fail("writing to repositories file", err)
	}

	for _, t := range tupStars {
		record := []string{t.name, fmt.Sprintf("%d", t.num)}
		if err := cw.Write(record); err != nil {
			fail("writing to repositories file", err)
		}
	}

	tupStars = nil

	for _, name := range noStars {
		record := []string{name, "0"}
		if err := cw.Write(record); err != nil {
			fail("writing to repositories file", err)
		}
	}

	cw.Flush()
	if err := cw.Error(); err != nil {
		fail("writing to repositories file", err)
	}
}
