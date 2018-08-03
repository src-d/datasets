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
	// ErrNoLicenseFound is raised if no license files were found.
	ErrNoLicenseFound = errors.New("no license file was found")

	globalLicenseDatabase = &LicenseDatabase{}

	// Base names of guessable license files.
	fileNames = []string{
		"copying",
		"copyleft",
		"copyright",
		"license",
		"unlicense",
		"licence",
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
		".html": PreprocessHTML,
	}

	licenseFileRe = regexp.MustCompile(
		fmt.Sprintf("^(%s)(%s)$",
			strings.Join(fileNames, "|"),
			strings.Replace(strings.Join(fileExtensions, "|"), ".", "\\.", -1)))

	readmeFileRe = regexp.MustCompile(fmt.Sprintf("^readme(%s)$",
		strings.Replace(strings.Join(fileExtensions, "|"), ".", "\\.", -1)))
)

// InvestigateProjectLicenses returns the most probable reference licenses matched for the given
// file tree. Each match has the confidence assigned, from 0 to 1, 1 means 100% confident.
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
	return InvestigateFilesLicenses(fileNames, func(file string) (string, error) {
		text, err := ioutil.ReadFile(paths.Join(path, file))
		return string(text), err
	})
}

// InvestigateFilesLicenses scans the given list of file names, reads them with `reader` and
// detects the licenses. Each match has the confidence assigned, from 0 to 1, 1 means 100% confident.
func InvestigateFilesLicenses(
	fileNames []string, reader func(string) (string, error)) (map[string]float32, error) {
	candidates := ExtractLicenseFiles(fileNames, reader)
	if len(candidates) == 0 {
		// Plan B: take the README, find the section about the license and apply NER
		candidates = ExtractReadmeFiles(fileNames, reader)
		if len(candidates) == 0 {
			return nil, ErrNoLicenseFound
		}
		licenses := InvestigateReadmeTexts(candidates)
		if len(licenses) == 0 {
			return nil, ErrNoLicenseFound
		}
		return licenses, nil
	}
	return InvestigateLicenseTexts(candidates), nil
}

// ExtractLicenseFiles returns the list of possible license texts.
// The file names are matched against the template.
// Reader is used to to read file contents.
func ExtractLicenseFiles(files []string, reader func(string) (string, error)) []string {
	candidates := []string{}
	for _, file := range files {
		if licenseFileRe.MatchString(strings.ToLower(file)) {
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

// InvestigateLicenseTexts takes the list of candidate license texts and returns the most probable
// reference licenses matched. Each match has the confidence assigned, from 0 to 1, 1 means 100% confident.
func InvestigateLicenseTexts(texts []string) map[string]float32 {
	maxLicenses := map[string]float32{}
	for _, text := range texts {
		candidates := InvestigateLicenseText(text)
		for name, sim := range candidates {
			maxSim := maxLicenses[name]
			if sim > maxSim {
				maxLicenses[name] = sim
			}
		}
	}
	return maxLicenses
}

// InvestigateLicenseText takes the license text and returns the most probable reference licenses matched.
// Each match has the confidence assigned, from 0 to 1, 1 means 100% confident.
func InvestigateLicenseText(text string) map[string]float32 {
	return globalLicenseDatabase.QueryLicenseText(text)
}

// ExtractReadmeFiles searches for README files.
// Reader is used to to read file contents.
func ExtractReadmeFiles(files []string, reader func(string) (string, error)) []string {
	candidates := []string{}
	for _, file := range files {
		if readmeFileRe.MatchString(strings.ToLower(file)) {
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

// InvestigateReadmeTexts scans README files for licensing information and outputs the
// probable names using NER.
func InvestigateReadmeTexts(texts []string) map[string]float32 {
	maxLicenses := map[string]float32{}
	for _, text := range texts {
		candidates := InvestigateReadmeText(text)
		for name, sim := range candidates {
			maxSim := maxLicenses[name]
			if sim > maxSim {
				maxLicenses[name] = sim
			}
		}
	}
	return maxLicenses
}

// InvestigateReadmeText scans the README file for licensing information and outputs probable
// names found with Named Entity Recognition from NLP.
func InvestigateReadmeText(text string) map[string]float32 {
	return globalLicenseDatabase.QueryReadmeText(text)
}

func init() {
	if os.Getenv("LICENSE_DEBUG") != "" {
		globalLicenseDatabase.Debug = true
	}
	globalLicenseDatabase.Load()
}
