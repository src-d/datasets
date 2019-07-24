package main

import (
	"archive/zip"
	"bytes"
	"compress/zlib"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/spf13/pflag"
	"github.com/src-d/go-borges"
	"github.com/src-d/go-borges/legacysiva"
	"github.com/xitongsys/parquet-go-source/local"
	"github.com/xitongsys/parquet-go/parquet"
	"github.com/xitongsys/parquet-go/writer"
	bblfsh "gopkg.in/bblfsh/client-go.v3"
	"gopkg.in/bblfsh/sdk.v2/uast/nodes/nodesproto"
	progress "gopkg.in/cheggaaa/pb.v1"
	"gopkg.in/src-d/enry.v1"
	"gopkg.in/src-d/enry.v1/data"
	"gopkg.in/src-d/go-billy.v4/osfs"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	ggio "gopkg.in/src-d/go-git.v4/utils/ioutil"
)

func parseFlags() (
	inputDirectory, bblfshEndpoint, outputDirectory, outputFormat string,
	languages map[string]struct{}, workers int, monitor bool) {

	var langsList []string
	pflag.StringVarP(&outputDirectory, "output", "o", "uast", "Output directory where to save the results.")
	pflag.StringSliceVarP(&langsList, "languages", "l", []string{"all"},
		"Programming languages to parse. The full list is at https://docs.sourced.tech/babelfish/languages Several "+
			"values can be specified separated by commas. The strings should be lower case. The special "+
			"value \"all\" disables any filtering. Example: --languages=c++,python")
	pflag.StringVarP(&outputFormat, "format", "f", "zip", "Output format: choose one of zip, parquet.")
	pflag.StringVarP(&bblfshEndpoint, "bblfsh", "b", "0.0.0.0:9432", "Babelfish server address.")
	pflag.IntVarP(&workers, "workers", "n", runtime.NumCPU()*2, "Number of goroutines to parse UASTs.")
	pflag.BoolVarP(&monitor, "monitor", "m", false, "Activate the advanced detection of \"bad\" " +
		"repositories and automatic restart on failures.")
	pflag.Parse()
	if pflag.NArg() != 1 {
		log.Fatalf("usage: pga2uast /path/to/directory/with/siva")
	}
	inputDirectory = pflag.Arg(0)
	languages = map[string]struct{}{}
	for _, lang := range langsList {
		if lang == "all" {
			languages["all"] = struct{}{}
			break
		}
		if canonical, exists := data.LanguageByAliasMap[lang]; !exists {
			log.Fatalf("language not supported: %s\n", lang)
		} else {
			languages[canonical] = struct{}{}
		}
	}
	if outputFormat != "zip" && outputFormat != "parquet" {
		log.Fatalf("unsupported output format: %s", outputFormat)
	}
	if err := os.MkdirAll(outputDirectory, 0777); err != nil {
		log.Fatalf("cannot initialize the output directory: %v", err)
	}
	return
}

const headRefPrefix = "refs/heads/HEAD/"

func listHeads(r *git.Repository) ([]plumbing.Hash, []string, error) {
	head, err := r.Head()
	if err == nil {
		return []plumbing.Hash{head.Hash()},
			[]string{strings.TrimPrefix(head.Name().String(), headRefPrefix)},
			nil
	}
	if err != plumbing.ErrReferenceNotFound {
		return nil, nil, err
	}
	refs, err := r.References()
	if err != nil {
		return nil, nil, err
	}
	var heads []plumbing.Hash
	var names []string
	err = refs.ForEach(func(ref *plumbing.Reference) error {
		if strings.HasPrefix(ref.Name().String(), headRefPrefix) {
			heads = append(heads, ref.Hash())
			names = append(names, strings.TrimPrefix(ref.Name().String(), headRefPrefix))
		}
		return nil
	})
	return heads, names, err
}

func readGitFile(f *object.File) (content []byte, err error) {
	reader, err := f.Reader()
	if err != nil {
		return nil, err
	}
	defer ggio.CheckClose(reader, &err)

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(reader); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func createProgressBar(repos borges.RepositoryIterator) *progress.ProgressBar {
	bar := progress.New(0)
	bar.Start()
	bar.ShowPercent = false
	bar.ShowSpeed = false
	bar.ShowElapsedTime = true
	bar.ShowFinalTime = false
	err := repos.ForEach(func(r borges.Repository) error {
		bar.Total++
		return nil
	})
	if err != nil {
		log.Fatalf("failed to iterate repositories: %v", err)
	}
	return bar
}

func compressBytes(buffer []byte) []byte {
	output := &bytes.Buffer{}
	zw := zlib.NewWriter(output)
	_, err := zw.Write(buffer)
	if err != nil {
		log.Panicf("compress/write: %v", err)
	}
	err = zw.Close()
	if err != nil {
		log.Panicf("compress/close: %v", err)
	}
	return output.Bytes()
}

func decompressBytes(buffer []byte) []byte {
	input := bytes.NewBuffer(buffer)
	zr, err := zlib.NewReader(input)
	if err != nil {
		log.Panicf("decompress/open: %v", err)
	}
	defer zr.Close()
	output, err := ioutil.ReadAll(zr)
	if err != nil {
		log.Panicf("decompress/read: %v", err)
	}
	return output
}

type parseTask struct {
	FileName string
	FullPath string
	CompressedContents []byte
	HeadUasts map[string][]byte
}

const BlacklistFileName = "blacklist.txt"

func getCurrentTaskFilePath() string {
	return filepath.Join(os.TempDir(), "pga2uast-current-task.txt")
}

func processRepository(
	r borges.Repository, bblfshEndpoint, outputDirectory, outputFormat string, languages map[string]struct{},
	workers int, bar *progress.ProgressBar, filesProcessed *int) (elapsed time.Duration) {

	startTime := time.Now()
	defer func() {
		elapsed = time.Now().Sub(startTime)
	}()
	defer bar.Increment()
	rid := r.ID().String()

	if blacklist, err := ioutil.ReadFile(BlacklistFileName); err == nil {
		for _, black := range strings.Split(string(blacklist), "\n") {
			if black == rid {
				log.Printf("skipped %s because it is blacklisted", rid)
				return
			}
		}
	}

	if _, err := os.Stat(getOutputFileName(outputDirectory, rid, outputFormat)); err == nil {
		return
	}
	if err := ioutil.WriteFile(getCurrentTaskFilePath(), []byte(rid), 0666); err != nil {
		log.Fatalf("cannot write %s: %v", getCurrentTaskFilePath(), err)
	}
	heads, names, err := listHeads(r.R())
	if len(heads) == 0 {
		log.Printf("%s: no heads: %v", rid, err)
		return
	}
	wg := sync.WaitGroup{}
	uasts := map[string]map[string][]byte{}
	headLock := sync.Mutex{}
	parseTasks := make(chan parseTask, workers*2)
	for i:=0; i<workers; i++ {
		go func() {
			client, err := bblfsh.NewClient(bblfshEndpoint)
			if err != nil {
				log.Panicf("cannot initialize the Bablefish client on %s: %v", bblfshEndpoint, err)
			}
			defer client.Close()
			for {
				task, more := <-parseTasks
				if !more {
					break
				}
				uast, err := parseFile(client, task.FullPath, decompressBytes(task.CompressedContents))
				if err == nil {
					headLock.Lock()
					task.HeadUasts[task.FileName] = compressBytes(uast)
					headLock.Unlock()
				}
				wg.Done()
			}
		}()
	}
	for headIndex, head := range heads {
		headUasts := map[string][]byte{}
		uasts[names[headIndex]] = headUasts
		commit, err := r.R().CommitObject(head)
		if err != nil {
			log.Printf("%s: no commit %s: %v", rid, head.String(), err)
			return
		}
		fileIter, err := commit.Files()
		if err != nil {
			log.Printf("%s: failed to list %s: %v", rid, head.String(), err)
			return
		}
		err = fileIter.ForEach(func(file *object.File) error {
			bin, err := file.IsBinary()
			if err != nil {
				return err
			}
			if bin {
				return nil
			}
			contents, err := readGitFile(file)
			if err != nil {
				log.Printf("%s: failed to read %s at %s: %v",
					rid, file.Name, file.Hash.String(), err)
				return err
			}
			if _, all := languages["all"]; !all {
				lang := enry.GetLanguage(file.Name, contents)
				if _, exists := languages[lang]; !exists {
					return nil
				}
			}
			wg.Add(1)
			parseTasks <- parseTask{
				FileName:           file.Name,
				FullPath:           fmt.Sprintf("%s/%s/%s", rid, names[headIndex], file.Name),
				CompressedContents: compressBytes(contents),
				HeadUasts:          headUasts,
			}
			*filesProcessed++
			bar.Postfix(fmt.Sprintf(" %s/%s %d", rid, names[headIndex], *filesProcessed))
			return nil
		})
		if err != nil {
			log.Printf("%s: failed to iter files in %s: %v", rid, head.String(), err)
		}
	}
	close(parseTasks)
	wg.Wait()
	if err := writeOutput(rid, uasts, outputDirectory, outputFormat); err != nil {
		log.Fatalf("failed to write the results for %s: %v", rid, err)
	}
	return
}

func printElapsedTimes(times map[string]time.Duration) {
	type pair struct {
		Duration time.Duration
		Name     string
	}
	var pairs []pair
	for k, v := range times {
		pairs = append(pairs, pair{v, k})
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Duration > pairs[j].Duration // reverse order
	})
	for _, p := range pairs {
		fmt.Printf("%s,%f\n", p.Name, float64(p.Duration)/1000000000)
	}
}

func parseFile(client *bblfsh.Client, path string, contents []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()
	request := client.NewParseRequest().
		Content(string(contents)).Filename(path).Mode(bblfsh.Semantic).Context(ctx)
	response, _, err := request.UAST()
	if err != nil {
		return nil, err
	}
	buf := &bytes.Buffer{}
	err = nodesproto.WriteTo(buf, response)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type parquetItem struct {
	Head string `parquet:"name=head, type=UTF8"`
	Path string `parquet:"name=path, type=UTF8"`
	UAST string `parquet:"name=uast, type=BYTE_ARRAY"`
}

func getOutputFileName(outputDirectory, repo, outputFormat string) string {
	return filepath.Join(outputDirectory, repo) + "." + outputFormat
}

func writeOutput(repo string, uasts map[string]map[string][]byte,
	outputDirectory, outputFormat string) (err error) {

	empty := true
	for _, v := range uasts {
		for range v {
			empty = false
			break
		}
		if !empty {
			break
		}
	}
	if empty {
		return nil
	}
	fileName := getOutputFileName(outputDirectory, repo, outputFormat)
	var file *os.File
	file, err = os.Create(fileName)
	if err != nil {
		return
	}
	defer func() {
		if cerr := file.Close(); err == nil {
			err = cerr
		}
	}()
	if outputFormat == "zip" {
		err = writeZIP(uasts, file)
		return
	} else if outputFormat == "parquet" {
		err = writeParquet(uasts, file)
		return
	} else {
		log.Panicf("unknown output format %s", outputFormat)
	}
	return nil
}

func writeZIP(uasts map[string]map[string][]byte, file *os.File) (err error) {
	zw := zip.NewWriter(file)
	defer func() {
		if cerr := zw.Close(); err == nil {
			err = cerr
		}
	}()
	for k, v := range uasts {
		for sk, uast := range v {
			uw, zerr := zw.Create(path.Join(k, sk))
			if zerr != nil {
				err = zerr
				return
			}
			_, err = uw.Write(decompressBytes(uast))
			if err != nil {
				return
			}
			v[sk] = nil  // free memory
		}
	}
	return
}

func writeParquet(uasts map[string]map[string][]byte, file *os.File) (err error) {
	lf := &local.LocalFile{FilePath: file.Name(), File: file}
	pw, err := writer.NewParquetWriter(lf, new(parquetItem), int64(runtime.NumCPU()))
	if err != nil {
		return
	}
	pw.CompressionType = parquet.CompressionCodec_GZIP
	for k, v := range uasts {
		for sk, uast := range v {
			if err = pw.Write(parquetItem{k, sk, string(decompressBytes(uast))}); err != nil {
				return
			}
			v[sk] = nil  // free memory
		}
	}
	err = pw.WriteStop()
	return
}

const SlaveEnvVar = "pga2uast-slave"

func launchSlave() *exec.Cmd {
	cmd := exec.Command(os.Args[0])
	e := os.Environ()
	e = append(e, SlaveEnvVar + "=1")
	cmd.Env = e
	for _, arg := range os.Args[1:] {
		if arg != "-m" && arg != "--monitor" {
			cmd.Args = append(cmd.Args, arg)
		}
	}
	log.Printf("running %s", strings.Join(cmd.Args, " "))
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start the slave process: %v", err)
	}
	return cmd
}

func becomeMonitor() {
	for {
		if err := launchSlave().Wait(); err == nil {
			break
		}
		task, err := ioutil.ReadFile(getCurrentTaskFilePath())
		if err != nil {
			log.Fatalf("cannot read %s: %v", getCurrentTaskFilePath(), err)
		}
		log.Printf("blacklisting %s", string(task))
		blacklist, _ := ioutil.ReadFile(BlacklistFileName)
		if len(blacklist) > 0 {
			blacklist = append(blacklist, byte('\n'))
		}
		blacklist = append(blacklist, task...)
		if err = ioutil.WriteFile(BlacklistFileName, blacklist, 0666); err != nil {
			log.Fatalf("cannot write %s: %v", BlacklistFileName, err)
		}
	}
}

func main() {
	inputDirectory, bblfshEndpoint, outputDirectory, outputFormat, languages, workers, monitor := parseFlags()
	if monitor {
		becomeMonitor()
		return
	}
	fs := osfs.New(inputDirectory)
	lib, err := legacysiva.NewLibrary("pga2siva", fs, &legacysiva.LibraryOptions{})
	if err != nil {
		log.Fatalf("legacysiva.NewLibrary failed: %v", err)
	}
	repos, err := lib.Repositories(borges.ReadOnlyMode)
	if err != nil {
		log.Fatalf("lib.Repositories failed: %v", err)
	}
	bar := createProgressBar(repos)
	defer bar.Finish()
	repos, _ = lib.Repositories(borges.ReadOnlyMode)
	filesProcessed := 0
	times := map[string]time.Duration{}
	defer printElapsedTimes(times)
	err = repos.ForEach(func(r borges.Repository) error {
		times[r.ID().String()] = processRepository(
			r, bblfshEndpoint, outputDirectory, outputFormat, languages, workers, bar, &filesProcessed)
		return nil
	})
	if err != nil {
		os.Exit(1)
	}
}
