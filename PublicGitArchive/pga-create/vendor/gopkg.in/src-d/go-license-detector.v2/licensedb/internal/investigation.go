package internal

import (
	"bytes"
	"fmt"
	paths "path"
	"regexp"
	"strings"

	"gopkg.in/src-d/go-license-detector.v2/licensedb/filer"
	"gopkg.in/src-d/go-license-detector.v2/licensedb/internal/processors"
)

var (
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

	licenseDirectoryRe = regexp.MustCompile(fmt.Sprintf(
		"^(%s)$", strings.Join(licenseFileNames, "|")))
)

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
func InvestigateReadmeTexts(texts [][]byte, fs filer.Filer) map[string]float32 {
	maxLicenses := map[string]float32{}
	for _, text := range texts {
		candidates := InvestigateReadmeText(text, fs)
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
func InvestigateReadmeText(text []byte, fs filer.Filer) map[string]float32 {
	return globalLicenseDatabase.QueryReadmeText(string(text), fs)
}

// IsLicenseDirectory indicates whether the directory is likely to contain licenses.
func IsLicenseDirectory(fileName string) bool {
	return licenseDirectoryRe.MatchString(strings.ToLower(fileName))
}
