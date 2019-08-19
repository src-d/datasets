package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
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
	inputDirectory, outputDirectory, outputFormat string, languages map[string]struct{}, workers int) {

	var langsList []string
	pflag.StringVarP(&outputDirectory, "output", "o", "files", "Output directory where to save the results.")
	pflag.StringSliceVarP(&langsList, "languages", "l", []string{"all"},
		"Programming languages to parse. The full list is at https://docs.sourced.tech/babelfish/languages Several "+
			"values can be specified separated by commas. The strings should be lower case. The special "+
			"value \"all\" disables any filtering. Example: --languages=c++,python")
	pflag.StringVarP(&outputFormat, "format", "f", "zip", "Output format: choose one of zip, parquet.")
	pflag.IntVarP(&workers, "workers", "n", runtime.NumCPU()*2, "Number of goroutines to read siva files.")
	pflag.Parse()
	if pflag.NArg() != 1 {
		log.Fatalf("usage: list_heads /path/to/directory/with/siva")
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

func getCurrentTaskFilePath() string {
	return filepath.Join(os.TempDir(), "pga2uast-current-task.txt")
}

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

func createProgressBar(repos borges.RepositoryIterator) *progress.ProgressBar {
	bar := progress.New(0)
	bar.Start()
	bar.ShowPercent = false
	bar.ShowSpeed = false
	bar.ShowElapsedTime = true
	bar.ShowFinalTime = false
	go func() {
		err := repos.ForEach(func(r borges.Repository) error {
			bar.Total++
			return nil
		})
		if err != nil {
			log.Fatalf("failed to iterate repositories: %v", err)
		}
	}()
	return bar
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

type parquetItem struct {
	Head string `parquet:"name=head, type=UTF8"`
	Path string `parquet:"name=path, type=UTF8"`
}

func getOutputFileName(outputDirectory, repo, outputFormat string) string {
	return filepath.Join(outputDirectory, repo) + "." + outputFormat
}

func writeOutput(repo string, files map[string][]string,
	outputDirectory, outputFormat string) (err error) {

	empty := true
	for _, v := range files {
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
	var outFile *os.File
	outFile, err = os.Create(fileName)
	if err != nil {
		return
	}
	defer func() {
		if cerr := outFile.Close(); err == nil {
			err = cerr
		}
	}()
	if outputFormat == "zip" {
		err = writeZIP(files, outFile)
		return
	} else if outputFormat == "parquet" {
		err = writeParquet(files, outFile)
		return
	} else {
		log.Panicf("unknown output format %s", outputFormat)
	}
	return nil
}

func writeZIP(files map[string][]string, file *os.File) (err error) {
	zw := zip.NewWriter(file)
	defer func() {
		if cerr := zw.Close(); err == nil {
			err = cerr
		}
	}()
	for k, v := range files {
		uw, zerr := zw.Create(k + ".txt")
		if zerr != nil {
			err = zerr
			return
		}
		for _, fp := range v {
			_, err = uw.Write([]byte(fp + "\n"))
			if err != nil {
				return
			}
		}
	}
	return
}

func writeParquet(files map[string][]string, file *os.File) (err error) {
	lf := &local.LocalFile{FilePath: file.Name(), File: file}
	pw, err := writer.NewParquetWriter(lf, new(parquetItem), int64(runtime.NumCPU()))
	if err != nil {
		return
	}
	pw.CompressionType = parquet.CompressionCodec_GZIP
	for k, v := range files {
		for _, fp := range v {
			if err = pw.Write(parquetItem{k, fp}); err != nil {
				return
			}
		}
	}
	err = pw.WriteStop()
	return
}

func processRepository(
	r borges.Repository, outputDirectory, outputFormat string,
	languages map[string]struct{}) (elapsed time.Duration) {

	startTime := time.Now()
	defer func() {
		elapsed = time.Now().Sub(startTime)
	}()
	rid := r.ID().String()
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
	files := map[string][]string{}
	for headIndex, head := range heads {
		name := names[headIndex]
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
			files[name] = append(files[name], file.Name)
			return nil
		})
		if err != nil {
			log.Printf("%s: failed to iter files in %s: %v", rid, head.String(), err)
		}
	}
	if err = writeOutput(rid, files, outputDirectory, outputFormat); err != nil {
		log.Printf("%s: failed to write the results: %v", rid, err)
	}
	return
}

func main() {
	inputDirectory, outputDirectory, outputFormat, languages, workers := parseFlags()
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
	times := map[string]time.Duration{}
	defer printElapsedTimes(times)
	timesLock := sync.Mutex{}
	jobs := make(chan borges.Repository, workers*2)
	wg := sync.WaitGroup{}
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			for r := range jobs {
				elapsed := processRepository(r, outputDirectory, outputFormat, languages)
				timesLock.Lock()
				times[r.ID().String()] = elapsed
				timesLock.Unlock()
			}
			wg.Done()
		}()
	}

	_ = repos.ForEach(func(r borges.Repository) error {
		jobs <- r
		bar.Add(1)
		return nil
	})
	close(jobs)
	wg.Wait()
}
