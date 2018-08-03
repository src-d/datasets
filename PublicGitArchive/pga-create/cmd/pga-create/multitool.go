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
	paths "path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/briandowns/spinner"
	"github.com/colinmarc/hdfs"
	"github.com/dustin/go-humanize"
	"github.com/pkg/errors"
	"golang.org/x/net/html"
	"gopkg.in/cheggaaa/pb.v1"
)

const (
	defaultGhtorrentMySQL = "http://ghtorrent-downloads.ewi.tudelft.nl/mysql/"
)

func fail(operation string, err error) {
	fmt.Fprintf(os.Stderr, "Error: %s: %s\n", operation, err.Error())
	os.Exit(1)
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

type selectionParameters struct {
	StarsFile         string
	LanguagesFile     string
	ReposFile         string
	FilteredLanguages []string
	MinStars          int
	TopN              int
	URLTemplate       string
}

func filterStars(path string, minStars int, topN int, selectedRepos map[int]bool) map[int]bool {
	f, err := os.Open(path)
	if err != nil {
		fail("opening stars file "+path, err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	repos := map[int]bool{}
	var stars int
	for scanner.Scan() {
		if len(repos) >= topN && topN > -1 {
			fmt.Fprintf(os.Stderr, "\rEffective ★ : %d%s\n", stars, strings.Repeat(" ", 40))
			break
		}
		line := scanner.Text()
		var repo int
		n, err := fmt.Sscan(line, &repo, &stars)
		if err != nil || n != 2 {
			if err == nil {
				err = errors.New("failed to parse " + line)
			}
			fail("parsing stars file "+path, err)
		}
		if selectedRepos != nil && !selectedRepos[repo] {
			continue
		}
		if stars >= minStars {
			repos[repo] = true
		} else {
			// the file is sorted
			break
		}
	}
	return repos
}

func filterLanguages(path string, languages []string) map[int]bool {
	gzf, err := os.Open(path)
	if err != nil {
		fail("opening languages file "+path, err)
	}
	defer gzf.Close()
	f, err := gzip.NewReader(gzf)
	if err != nil {
		fail("decompressing languages file "+path, err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	langMap := map[string]bool{}
	for _, lang := range languages {
		langMap[lang] = true
	}
	result := map[int]bool{}
	active := false
	for scanner.Scan() {
		line := scanner.Text()
		if line[0] == '#' {
			active = langMap[line[2:]]
			continue
		}
		if !active {
			continue
		}
		id, err := strconv.Atoi(line)
		if err != nil {
			fail("parsing languages file "+path+": "+line, err)
		}
		result[id] = true
	}
	return result
}

func selectRepos(parameters selectionParameters) {
	spin := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
	spin.Writer = os.Stderr
	spin.Start()

	var selectedRepos map[int]bool
	if len(parameters.FilteredLanguages) > 0 {
		spin.Suffix = " reading " + parameters.LanguagesFile
		selectedRepos = filterLanguages(parameters.LanguagesFile, parameters.FilteredLanguages)
	}
	spin.Suffix = " reading " + parameters.StarsFile
	selectedRepos = filterStars(
		parameters.StarsFile, parameters.MinStars, parameters.TopN, selectedRepos)
	spin.Stop()
	bar := pb.New(len(selectedRepos))
	bar.Output = os.Stderr
	bar.ShowFinalTime = true
	bar.ShowPercent = false
	bar.ShowSpeed = false
	bar.SetMaxWidth(80)
	bar.Start()
	defer bar.Finish()
	f, err := os.Open(parameters.ReposFile)
	if err != nil {
		fail("opening repositories file "+parameters.ReposFile, err)
	}
	defer f.Close()
	gzf, err := gzip.NewReader(f)
	if err != nil {
		fail("decompressing repositories file "+parameters.ReposFile, err)
	}
	defer gzf.Close()
	scanner := bufio.NewScanner(gzf)
	for scanner.Scan() {
		var repoID int
		var repoName string
		line := scanner.Text()
		n, err := fmt.Sscan(line, &repoID, &repoName)
		if err != nil || n != 2 {
			if err == nil {
				err = errors.New("failed to parse " + line)
			}
			fail("parsing repositories file "+parameters.ReposFile, err)
		}
		if selectedRepos[repoID] {
			bar.Increment()
			fmt.Fprintf(os.Stdout, parameters.URLTemplate+"\n", repoName)
		}
	}
}

func downloadIndex(baseURL string, outputPath string) {
	spin := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
	spin.Writer = os.Stderr
	spin.Start()
	defer spin.Stop()
	indexURL, err := url.Parse(baseURL)
	if err != nil {
		fail("parsing "+baseURL, err)
	}
	latestURL, _ := url.Parse("csv/latest.csv.gz")
	effectiveURL := indexURL.ResolveReference(latestURL).String()
	response, err := http.Get(effectiveURL)
	if err != nil {
		fail("connecting to "+effectiveURL, err)
	}
	defer response.Body.Close()
	var totalRead int64
	inputFile := trackingReader{RealReader: response.Body, Callback: func(n int) {
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
	var outputFile *os.File
	if outputPath == "-" {
		outputFile = os.Stdout
	} else {
		outputFile, err = os.Create(outputPath)
		if err != nil {
			fail("creating "+outputPath, err)
		}
	}
	bytesWritten, err := io.Copy(outputFile, gzf)
	spin.Stop()
	if err != nil {
		fail("failed to download "+effectiveURL, err)
	}
	fmt.Fprintf(os.Stderr, "Read      %s\nWrote     %s\n",
		humanize.Bytes(uint64(totalRead)), humanize.Bytes(uint64(bytesWritten)))
}

type downloadBackend interface {
	NewState() interface{}
	Download(state interface{}, response *http.Response)
}

type downloadManager struct {
	urls      chan *url.URL
	workers   int
	waitGroup *sync.WaitGroup
	backend   downloadBackend
	callback  func()
}

func (manager *downloadManager) Wait() {
	for i := 0; i < manager.workers; i++ {
		manager.urls <- nil
	}
	manager.waitGroup.Wait()
}

func (manager *downloadManager) work() {
	state := manager.backend.NewState()
	for {
		job := <-manager.urls
		if job == nil {
			manager.waitGroup.Done()
			break
		}
		manager.callback()
		response, err := http.Get(job.String())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to connect to %s\n", job.String())
			continue
		}
		if response.StatusCode != 200 {
			fmt.Fprintf(os.Stderr, "\n%s: HTTP %d\n", job.String(), response.StatusCode)
			continue
		}
		func() {
			defer response.Body.Close()
			manager.backend.Download(state, response)
		}()
	}
}

func newDownloadManager(
	urls chan *url.URL, workers int, backend downloadBackend, callback func()) *downloadManager {
	manager := &downloadManager{
		urls:      urls,
		workers:   workers,
		waitGroup: &sync.WaitGroup{},
		backend:   backend,
		callback:  callback,
	}
	manager.waitGroup.Add(workers)
	for i := 0; i < workers; i++ {
		go manager.work()
	}
	return manager
}

type fileSystemDownloadBackend struct {
	OutputPath string
}

func (backend *fileSystemDownloadBackend) NewState() interface{} {
	return nil
}

func (backend *fileSystemDownloadBackend) Download(state interface{}, response *http.Response) {
	job := response.Request.URL
	length := response.ContentLength
	filePath := paths.Join(backend.OutputPath, job.Path)
	file, err := os.Open(filePath)
	if err == nil {
		fi, err := file.Stat()
		if err == nil {
			if fi.Size() == length {
				return
			}
		}
	}
	err = os.MkdirAll(paths.Dir(filePath), os.ModePerm)
	if err != nil {
		fail("failed to create directories: "+paths.Dir(filePath), err)
	}
	file, err = os.Create(filePath)
	if err != nil {
		fail("failed to create "+filePath, err)
	}
	written, err := io.Copy(file, response.Body)
	if err != nil {
		fail("failed to download "+job.String(), err)
	}
	if written != length {
		fail("incomplete download: "+job.String(), fmt.Errorf("%d != %d", written, length))
	}
}

type hdfsDownloadBackend struct {
	Address    string
	OutputPath string
}

func (backend *hdfsDownloadBackend) NewState() interface{} {
	if strings.HasPrefix(backend.OutputPath, "hdfs://") {
		panic("OutputPath must be without hdfs://")
	}
	client, err := hdfs.New(backend.Address)
	if err != nil {
		fail("connecting to HDFS at "+backend.Address, err)
	}
	return client
}

func (backend *hdfsDownloadBackend) Download(state interface{}, response *http.Response) {
	job := response.Request.URL
	client := state.(*hdfs.Client)
	length := response.ContentLength
	filePath := paths.Join(backend.OutputPath, job.Path)
	fi, err := client.Stat(filePath)
	if err == nil {
		if fi.Size() == length {
			return
		}
	}
	err = client.MkdirAll(paths.Dir(filePath), os.ModePerm)
	if err != nil {
		fail("failed to create directories: "+paths.Dir(filePath), err)
	}
	_ = client.Remove(filePath)
	file, err := client.Create(filePath)
	if err != nil {
		fail("failed to create "+filePath, err)
	}
	defer file.Close()
	written, err := io.Copy(file, response.Body)
	if err != nil {
		fail("failed to download "+job.String(), err)
	}
	if written != length {
		fail("incomplete download: "+job.String(), fmt.Errorf("%d != %d", written, length))
	}
}

func downloadDataset(baseURL string, series string, outPath string, workers int, hdfs string) {
	base, err := url.Parse(baseURL)
	if err != nil {
		fail("parsing "+baseURL, err)
	}
	bar := pb.New(0)
	bar.ShowFinalTime = true
	bar.ShowPercent = false
	bar.ShowSpeed = false
	bar.SetMaxWidth(80)
	bar.Start()
	defer bar.Finish()
	totalUrls := 0
	finishedUrls := 0
	urls := make(chan *url.URL, 1<<24)
	defer close(urls)
	var backend downloadBackend
	if strings.HasPrefix(outPath, "hdfs://") {
		backend = &hdfsDownloadBackend{
			Address:    hdfs,
			OutputPath: outPath[7:],
		}
	} else {
		backend = &fileSystemDownloadBackend{
			OutputPath: outPath,
		}
	}
	downloader := newDownloadManager(urls, workers, backend, func() {
		finishedUrls++
		bar.Set(finishedUrls)
	})
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		fileURL, err := url.Parse(fmt.Sprintf("siva/%s/%s/%s", series, line[:2], line))
		if err != nil {
			fail("failed to parse file names: "+scanner.Text(), err)
		}
		totalUrls++
		bar.Total = int64(totalUrls)
		urls <- base.ResolveReference(fileURL)
	}
	downloader.Wait()
}

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

type selectCommand struct {
	Stars           string `short:"s" long:"stars" required:"true" description:"Input path for the file with the numbers of stars per repository."`
	Languages       string `short:"g" long:"languages" description:"Input path for the gzipped file with the mapping between languages and repositories."`
	Repositories    string `short:"r" long:"repositories" required:"true" description:"Input path for the gzipped file with the repository names and identifiers."`
	MinStars        int    `short:"m" long:"min-stars" description:"Minimum number of stars."`
	Max             int    `short:"n" long:"max" default:"-1" description:"Maximum number of top-starred repositories to clone. -1 means unlimited. Language filter is applied before."`
	FilterLanguages string `short:"l" long:"filter-languages" description:"Comma separated list of languages."`
	UrlTemplate     string `long:"url-template" default:"git://github.com/%s.git" description:"Output URL printf template."`
}

func (c *selectCommand) Execute(args []string) error {
	var filteredLangsSplitted []string
	if len(c.FilterLanguages) == 0 {
		filteredLangsSplitted = []string{}
	} else {
		filteredLangsSplitted = strings.Split(c.FilterLanguages, ",")
	}

	selectRepos(selectionParameters{
		StarsFile:         c.Stars,
		LanguagesFile:     c.Languages,
		ReposFile:         c.Repositories,
		MinStars:          c.MinStars,
		FilteredLanguages: filteredLangsSplitted,
		TopN:              c.Max,
		URLTemplate:       c.UrlTemplate,
	})

	return nil
}

type getIndexCommand struct {
	Base   string `short:"b" long:"base" default:"http://pga.sourced.tech/" description:"Base URL for index files."`
	Output string `short:"o" long:"output" default:"-" description:"Output path. \"-\" means stdout (default)."`
}

func (c *getIndexCommand) Execute(args []string) error {
	downloadIndex(c.Base, c.Output)
	return nil
}

type getDatasetCommand struct {
	Output  string `short:"o" long:"output" description:"Output directory. Required. If starts with \"hdfs://\", files are written to HDFS."`
	Base    string `short:"b" long:"base" default:"http://pga.sourced.tech/" description:"Base URL for Siva files."`
	Series  string `short:"s" long:"series" default:"latest" description:"Snapshot name."`
	Workers int    `short:"j" long:"workers" default:"10" description:"The number of parallel goroutines to run to download files."`
	HDFS    string `long:"hdfs" description:"HDFS namenode address:port."`
}

func (c *getDatasetCommand) Execute(args []string) error {
	downloadDataset(c.Base, c.Series, c.Output, c.Workers, c.HDFS)
	return nil
}
