package main

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/pkg/errors"
	pb "gopkg.in/cheggaaa/pb.v1"
)

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

type selectionParameters struct {
	StarsFile         string
	LanguagesFile     string
	ReposFile         string
	FilteredLanguages []string
	MinStars          int
	TopN              int
	URLTemplate       string
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
