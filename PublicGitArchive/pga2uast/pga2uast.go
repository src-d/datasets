package main

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
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
	"gopkg.in/src-d/go-git.v4/utils/ioutil"
)

func parseFlags() (
	inputDirectory, bblfshEndpoint, outputDirectory, outputFormat string,
	languages map[string]struct{}) {

	var langsList []string
	pflag.StringVarP(&outputDirectory, "output", "o", "uast", "Output directory where to save the results.")
	pflag.StringSliceVarP(&langsList, "languages", "l", []string{"all"},
		"Programming languages to parse. The full list is at https://docs.sourced.tech/babelfish/languages Several "+
			"values can be specified separated by commas. The strings should be lower case. The special "+
			"value \"all\" disables any filtering. Example: --languages=c++,python")
	pflag.StringVarP(&outputFormat, "format", "f", "zip", "Output format: choose one of zip, parquet.")
	pflag.StringVarP(&bblfshEndpoint, "bblfsh", "b", "0.0.0.0:9432", "Babelfish server address.")
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
	defer ioutil.CheckClose(reader, &err)

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

func processRepository(
	r borges.Repository, bblfshEndpoint, outputDirectory, outputFormat string, languages map[string]struct{},
	bar *progress.ProgressBar, filesProcessed *int) (elapsed time.Duration) {

	startTime := time.Now()
	defer func() {
		elapsed = time.Now().Sub(startTime)
	}()
	defer bar.Increment()
	rid := r.ID().String()
	if _, err := os.Stat(filepath.Join(outputDirectory, rid) + "." + outputFormat); err == nil {
		return
	}
	heads, names, err := listHeads(r.R())
	if len(heads) == 0 {
		log.Printf("%s: no heads: %v", rid, err)
		return
	}
	wg := sync.WaitGroup{}
	uasts := map[string]map[string][]byte{}
	headLock := sync.Mutex{}
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
			go func(fileName string, contents []byte, headUasts map[string][]byte) {
				defer wg.Done()
				uast, err := parseFile(bblfshEndpoint, fileName, contents)
				if err == nil {
					headLock.Lock()
					headUasts[file.Name] = uast
					headLock.Unlock()
				}
			}(fmt.Sprintf("%s/%s/%s", rid, names[headIndex], file.Name), contents, headUasts)
			*filesProcessed++
			bar.Postfix(fmt.Sprintf(" %s/%s %d", rid, names[headIndex], *filesProcessed))
			return nil
		})
		if err != nil {
			log.Printf("%s: failed to iter files in %s: %v", rid, head.String(), err)
		}
	}
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

func parseFile(endpoint string, path string, contents []byte) ([]byte, error) {
	client, err := bblfsh.NewClient(endpoint)
	if err != nil {
		return nil, err
	}
	defer client.Close()
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
	fileName := filepath.Join(outputDirectory, repo) + "." + outputFormat
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
		panic(fmt.Sprintf("unknown output format %s", outputFormat))
	}
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
			_, err = uw.Write(uast)
			if err != nil {
				return
			}
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
			if err = pw.Write(parquetItem{k, sk, string(uast)}); err != nil {
				return
			}
		}
	}
	err = pw.WriteStop()
	return
}

func main() {
	inputDirectory, bblfshEndpoint, outputDirectory, outputFormat, languages := parseFlags()
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
			r, bblfshEndpoint, outputDirectory, outputFormat, languages, bar, &filesProcessed)
		return nil
	})
	if err != nil {
		os.Exit(1)
	}
}
