package main

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/dustin/go-humanize"
	"github.com/pkg/errors"
	"golang.org/x/net/html"
)

const (
	defaultGhtorrentMySQL = "http://ghtorrent-downloads.ewi.tudelft.nl/mysql/"
)

type discoverCommand struct {
	URL          string `short:"l" long:"url" description:"Link to GHTorrent MySQL dump in tar.gz format. If \"-\", stdin is read. If empty (default), find the most recent dump at GHTORRENT_MYSQL ?= http://ghtorrent-downloads.ewi.tudelft.nl/mysql/."`
	Stars        string `short:"s" long:"stars" required:"true" description:"Output path for the file with the numbers of stars per repository."`
	Languages    string `short:"g" long:"languages" description:"Output path for the gzipped file with the mapping between languages and repositories. May be empty - will be skipped then."`
	Repositories string `short:"r" long:"repositories" required:"true" description:"Output path for the gzipped file with the repository names and identifiers."`
}

func (c *discoverCommand) Execute(args []string) error {
	discoverRepos(discoveryParameters{
		URL:           c.URL,
		StarsPath:     c.Stars,
		LanguagesPath: c.Languages,
		ReposPath:     c.Repositories,
	})

	return nil
}

type discoveryParameters struct {
	URL           string
	StarsPath     string
	LanguagesPath string
	ReposPath     string
}

type trackingReader struct {
	RealReader io.Reader
	Callback   func(n int)
}

func (reader trackingReader) Read(p []byte) (n int, err error) {
	n, err = reader.RealReader.Read(p)
	reader.Callback(n)
	return n, err
}

func reduceWatchers(stream io.Reader) map[int]int {
	stars := map[int]int{}
	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		line := scanner.Text()
		commaPos := strings.Index(line, ",")
		projectID, err := strconv.Atoi(line[:commaPos])
		if err != nil {
			fail(fmt.Sprintf("parsing watchers project ID \"%s\"", line[:commaPos]), err)
		}
		stars[projectID]++
	}
	return stars
}

type watchPair struct {
	Value int
	Key   int
}

func writeWatchers(stars map[int]int, ids map[int][]int, path string) {
	pairs := make([]watchPair, 0, len(stars))
	majors := map[int]bool{}
	for key := range stars {
		myids := ids[key]
		// if it is a reference, go to the main repository
		if myids[0] >= 0 {
			key = myids[0]
			myids = ids[key]
		}
		if majors[key] {
			continue
		}
		majors[key] = true
		maxval := stars[key]
		for _, id := range myids[1:] {
			other := stars[id]
			if other > maxval {
				maxval = other
			}
		}
		pairs = append(pairs, watchPair{Key: key, Value: maxval})
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Value > pairs[j].Value // reverse order
	})
	f, err := os.Create(path)
	defer f.Close()
	if err != nil {
		fail("creating watchers file "+path, err)
	}
	for _, pair := range pairs {
		fmt.Fprintf(f, "%d %d\n", pair.Key, pair.Value)
	}
}

func writeProjects(stream io.Reader, path string) map[int][]int {
	f, err := os.Create(path)
	defer f.Close()
	if err != nil {
		fail("creating repositories file "+path, err)
	}
	gzf := gzip.NewWriter(f)
	defer gzf.Close()
	scanner := bufio.NewScanner(stream)
	skip := false
	projects := map[string]int{}
	ids := map[int][]int{}
	for scanner.Scan() {
		line := scanner.Text()
		skipThis := skip
		skip = line[len(line)-1:] == "\\"
		if skipThis {
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
		// there can be duplicates
		// the slice is always at least one element
		// negative value means the original (main) repository
		// otherwise it is a reference to the main one
		if existingID, exists := projects[projectName]; !exists {
			projects[projectName] = projectID
			ids[projectID] = make([]int, 1, 1)
			ids[projectID][0] = -1
			fmt.Fprintf(gzf, "%d %s\n", projectID, projectName)
		} else {
			ids[existingID] = append(ids[existingID], projectID)
			ids[projectID] = make([]int, 1, 1)
			ids[projectID][0] = existingID
		}
	}
	return ids
}

func writeLanguages(stream io.Reader, path string) {
	f, err := os.Create(path)
	defer f.Close()
	if err != nil {
		fail("creating languages file "+path, err)
	}
	gzf := gzip.NewWriter(f)
	defer gzf.Close()
	scanner := bufio.NewScanner(stream)
	langs := map[string]map[int]bool{}
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.SplitN(line, ",", 3)
		projectID, err := strconv.Atoi(fields[0])
		if err != nil {
			fail("parsing project_languages.csv: "+line, err)
		}
		lang := fields[1][1 : len(fields[1])-1]
		projects := langs[lang]
		if projects == nil {
			projects = map[int]bool{}
			langs[lang] = projects
		}
		projects[projectID] = true
	}
	langList := make([]string, len(langs))
	{
		i := 0
		for lang := range langs {
			langList[i] = lang
			i++
		}
	}
	sort.Strings(langList)
	for _, lang := range langList {
		projects := langs[lang]
		projectsList := make([]int, len(projects))
		{
			i := 0
			for id := range projects {
				projectsList[i] = id
				i++
			}
		}
		sort.Ints(projectsList)
		fmt.Fprintf(gzf, "# %s\n", lang)
		for _, id := range projectsList {
			fmt.Fprintln(gzf, id)
		}
	}
}

func findMostRecentMySQLDump(root string) string {
	ghturl, err := url.Parse(root)
	if err != nil {
		fail("parsing "+root, err)
	}
	response, err := http.Get(ghturl.String())
	if err != nil {
		fail("connecting to "+ghturl.String(), err)
	}
	defer response.Body.Close()
	tokenizer := html.NewTokenizer(response.Body)
	dumps := []string{}
	for token := tokenizer.Next(); token != html.ErrorToken; token = tokenizer.Next() {
		if token == html.StartTagToken {
			tag := tokenizer.Token()
			if tag.Data == "a" {
				for _, attr := range tag.Attr {
					if attr.Key == "href" {
						dumps = append(dumps, attr.Val)
						break
					}
				}
			}
		}
	}
	if len(dumps) == 0 {
		fail("getting the list of available dumps", errors.New("no dumps found"))
	}
	sort.Strings(dumps)
	lastDumpStr := dumps[len(dumps)-1]
	dumpurl, err := url.Parse(lastDumpStr)
	if err != nil {
		fail("parsing "+lastDumpStr, err)
	}
	return ghturl.ResolveReference(dumpurl).String()
}

func discoverRepos(params discoveryParameters) {
	startTime := time.Now()
	spin := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
	spin.Start()
	defer spin.Stop()
	var inputFile io.Reader
	if params.URL == "-" {
		inputFile = os.Stdin
	} else {
		if params.URL == "" {
			envURL := os.Getenv("GHTORRENT_MYSQL")
			if envURL == "" {
				envURL = defaultGhtorrentMySQL
			}
			spin.Suffix = " " + envURL
			params.URL = findMostRecentMySQLDump(envURL)
			fmt.Printf("\r>> %s\n", params.URL)
			spin.Suffix = " connecting..."
		}
		response, err := http.Get(params.URL)
		if err != nil {
			fail("starting the download of "+params.URL, err)
		}
		inputFile = response.Body
		defer response.Body.Close()
	}
	var totalRead int64
	inputFile = trackingReader{RealReader: inputFile, Callback: func(n int) {
		totalRead += int64(n)
		if totalRead%3 == 0 {
			// supposing that mod 3 is random, this is a quick and dirty way to (/ 3) updates.
			spin.Suffix = fmt.Sprintf(" %s", humanize.Bytes(uint64(totalRead)))
		}
	}}
	gzf, err := gzip.NewReader(inputFile)
	if err != nil {
		fail("opening gzip stream", err)
	}
	defer gzf.Close()
	tarf := tar.NewReader(gzf)
	processed := int64(0)
	numTasks := 3
	if params.LanguagesPath == "" {
		numTasks--
	}
	status := 0
	i := 0
	var stars map[int]int
	var ids map[int][]int
	for header, err := tarf.Next(); err != io.EOF; header, err = tarf.Next() {
		if err != nil {
			fail("reading tar.gz", err)
		}
		i++
		processed += header.Size
		isWatchers := strings.HasSuffix(header.Name, "watchers.csv")
		isProjects := strings.HasSuffix(header.Name, "projects.csv")
		isLanguages := strings.HasSuffix(header.Name, "project_languages.csv")
		mark := " "
		if isWatchers || isProjects || isLanguages {
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
			stars = reduceWatchers(tarf)
			status++
		} else if isProjects {
			ids = writeProjects(tarf, params.ReposPath)
			status++
		} else if isLanguages && params.LanguagesPath != "" {
			writeLanguages(tarf, params.LanguagesPath)
			status++
		}
		if stars != nil && ids != nil {
			writeWatchers(stars, ids, params.StarsPath)
		}
		if status == numTasks {
			break
		}
	}
	fmt.Printf("\nRead      %s\nProcessed %s\nElapsed   %s\n",
		humanize.Bytes(uint64(totalRead)), humanize.Bytes(uint64(processed)), time.Since(startTime))
}
