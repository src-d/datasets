package licensedb

import (
	"errors"
	"fmt"
	paths "path"
	"regexp"
	"strings"

	"bytes"
	"gopkg.in/src-d/go-license-detector.v2/licensedb/filer"
	"gopkg.in/src-d/go-license-detector.v2/licensedb/internal/processors"
)

var (
	// ErrNoLicenseFound is raised if no license files were found.
	ErrNoLicenseFound = errors.New("no license file was found")

	globalLicenseDatabase = loadLicenses()

	// Base names of guessable license files
	licenseFileNames = []string{
		"li[cs]en[cs]e(s?)",
		"legal",
		"copy(left|right|ing)",
		"unlicense",
		"l?gpl([-_ v]?)(\\d\\.?\\d)?",
		"bsd",
		"mit",
		"apache",
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

	filePreprocessors = map[string]func([]byte) []byte{
		".md":   processors.Markdown,
		".rst":  processors.RestructuredText,
		".html": processors.HTML,
	}

	licenseFileRe = regexp.MustCompile(
		fmt.Sprintf("^(|.*[-_. ])(%s)(|[-_. ].*)$",
			strings.Join(licenseFileNames, "|")))

	readmeFileRe = regexp.MustCompile(fmt.Sprintf("^(readme|guidelines)(%s)$",
		strings.Replace(strings.Join(fileExtensions, "|"), ".", "\\.", -1)))

	pureLicenseFileRe = regexp.MustCompile(fmt.Sprintf(
		"^(%s)$", strings.Join(licenseFileNames, "|")))
)

// Detect returns the most probable reference licenses matched for the given
// file tree. Each match has the confidence assigned, from 0 to 1, 1 means 100% confident.
func Detect(fs filer.Filer) (map[string]float32, error) {
	files, err := fs.ReadDir("")
	if err != nil {
		return nil, err
	}
	fileNames := []string{}
	for _, file := range files {
		if !file.IsDir {
			fileNames = append(fileNames, file.Name)
		} else if pureLicenseFileRe.MatchString(strings.ToLower(file.Name)) {
			// "license" directory, let's look inside
			subfiles, err := fs.ReadDir(file.Name)
			if err == nil {
				for _, subfile := range subfiles {
					if !subfile.IsDir {
						fileNames = append(fileNames, paths.Join(file.Name, subfile.Name))
					}
				}
			}
		}
	}
	candidates := ExtractLicenseFiles(fileNames, fs)
	licenses := InvestigateLicenseTexts(candidates)
	if len(licenses) > 0 {
		return licenses, nil
	}
	// Plan B: take the README, find the section about the license and apply NER
	candidates = ExtractReadmeFiles(fileNames, fs)
	if len(candidates) == 0 {
		return nil, ErrNoLicenseFound
	}
	licenses = InvestigateReadmeTexts(candidates)
	if len(licenses) == 0 {
		return nil, ErrNoLicenseFound
	}
	return licenses, nil
}

// ExtractLicenseFiles returns the list of possible license texts.
// The file names are matched against the template.
// Reader is used to to read file contents.
func ExtractLicenseFiles(files []string, fs filer.Filer) [][]byte {
	candidates := [][]byte{}
	for _, file := range files {
		if licenseFileRe.MatchString(strings.ToLower(paths.Base(file))) {
			text, err := fs.ReadFile(file)
			if len(text) < 128 {
				// e.g. https://github.com/Unitech/pm2/blob/master/LICENSE
				realText, err := fs.ReadFile(string(bytes.TrimSpace(text)))
				if err == nil {
					file = string(bytes.TrimSpace(text))
					text = realText
				}
			}
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
func InvestigateLicenseTexts(texts [][]byte) map[string]float32 {
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
func InvestigateLicenseText(text []byte) map[string]float32 {
	return globalLicenseDatabase.QueryLicenseText(string(text))
}

// ExtractReadmeFiles searches for README files.
// Reader is used to to read file contents.
func ExtractReadmeFiles(files []string, fs filer.Filer) [][]byte {
	candidates := [][]byte{}
	for _, file := range files {
		if readmeFileRe.MatchString(strings.ToLower(file)) {
			text, err := fs.ReadFile(file)
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
func InvestigateReadmeTexts(texts [][]byte) map[string]float32 {
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
func InvestigateReadmeText(text []byte) map[string]float32 {
	return globalLicenseDatabase.QueryReadmeText(string(text))
}
