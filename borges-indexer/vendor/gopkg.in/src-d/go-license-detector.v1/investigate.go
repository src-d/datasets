package ld

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	paths "path"
	"regexp"
	"strings"
)

var (
	NoLicenseFoundError = errors.New("no license file was found")

	globalLicenseDatabase = &LicenseDatabase{}
	// Base names of guessable license files.
	fileNames = []string{
		"copying",
		"copyleft",
		"copyright",
		"license",
		"unlicense",
	}

	// License file extensions. Combined with the fileNames slice
	// to create a set of files we can reasonably assume contain
	// licensing information.
	fileExtensions = []string{
		"",
		".md",
		".rst",
		".html",
		".txt",
	}

	filePreprocessors = map[string]func(string) string{
		".md":   PreprocessMarkdown,
		".rst":  PreprocessRestructuredText,
		".html": PreprocessHtml,
	}

	fileRe = regexp.MustCompile(
		fmt.Sprintf("^(%s)(%s)$",
			strings.Join(fileNames, "|"),
			strings.Replace(strings.Join(fileExtensions, "|"), ".", "\\.", -1)))
)

func InvestigateProjectLicenses(path string) (map[string]float32, error) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	fileNames := []string{}
	for _, file := range files {
		if !file.IsDir() {
			fileNames = append(fileNames, file.Name())
		}
	}
	candidates := ExtractLicenseFiles(fileNames, func(file string) (string, error) {
		text, err := ioutil.ReadFile(paths.Join(path, file))
		return string(text), err
	})
	if len(candidates) == 0 {
		return nil, NoLicenseFoundError
	}
	return InvestigateFiles(candidates), nil
}

func ExtractLicenseFiles(files []string, reader func(string) (string, error)) []string {
	candidates := []string{}
	for _, file := range files {
		if fileRe.MatchString(strings.ToLower(file)) {
			text, err := reader(file)
			if err == nil {
				if preprocessor, exists := filePreprocessors[paths.Ext(file)]; exists {
					text = preprocessor(text)
				}
				candidates = append(candidates, text)
			}
		}
	}
	return candidates
}

func InvestigateFiles(texts []string) map[string]float32 {
	maxLicenses := map[string]float32{}
	for _, text := range texts {
		options, similarities := InvestigateFile(text)
		for i, sim := range similarities {
			maxSim := maxLicenses[options[i]]
			if sim > maxSim {
				maxLicenses[options[i]] = sim
			}
		}
	}
	return maxLicenses
}

func InvestigateFile(text string) (options []string, similarities []float32) {
	return globalLicenseDatabase.Query(text)
}

func init() {
	if os.Getenv("LICENSENG_DEBUG") != "" {
		globalLicenseDatabase.Debug = true
	}
	globalLicenseDatabase.Load()
}
