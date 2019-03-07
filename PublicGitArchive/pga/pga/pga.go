// Package pga provides a simple API to access the Public Git Archive repository.
// For more information check http://pga.sourced.tech/.
package pga

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Repository contains the data used to filter the index and download files.
type Repository struct {
	URL       string   `json:"url"`           // URL of the repository.
	Filenames []string `json:"sivaFilenames"` // Siva filenames.
	Size      int64    `json:"size"`          // Sum of the siva files sizes.
	License   string   `json:"license"`       // Main license name of the repository.

	// Stats per language
	Languages             []string `json:"langs"`             // Languages found in the repository.
	LanguagesByteCount    []int64  `json:"langsByteCount"`    // Number of bytes for the language in same index.
	LanguagesLineCount    []int64  `json:"langsLinesCount"`   // Number of lines of code for the language in same index.
	LanguagesFileCount    []int64  `json:"langsFilesCount"`   // Number of files in the language in the same index.
	LanguagesEmptyLines   []int64  `json:"emptyLinesCount"`   // Number of blank lines for the language in the same index
	LanguagesCodeLines    []int64  `json:"codeLinesCount"`    // Number of lines of code for the language in the same index.
	LanguagesCommentLines []int64  `json:"commentLinesCount"` // Number of comment lines for the language in the same index.

	// Global stats
	Files    int64 `json:"fileCount"`     // Number of files in the repository.
	Commits  int64 `json:"commitsCount"`  // Number of commits in the repository.
	Branches int64 `json:"branchesCount"` // Number of branches in the repository.
	Forks    int64 `json:"forkCount"`     // Number of forks of this repository.
	Stars    int64 `json:"stars"`         // Number of stars of this repository.
}

// CSVHeaders are the headers expected for the CSV formatted index.
func CSVHeaders() []string {
	cp := make([]string, len(csvHeaders))
	copy(cp, csvHeaders)
	return cp
}

const (
	headerURL = iota
	headerSivaFilenames
	headerFileCount
	headerLangs
	headerLangsByteCount
	headerLangsLinesCount
	headerLangsFilesCount
	headerCommitsCount
	headerBranchesCount
	headerForkCount
	headerEmptyLinesCount
	headerCodeLinesCount
	headerCommentLinesCount
	headerLicense
	headerStars
	headerSize
)

var csvHeaders = []string{
	headerURL:               "URL",
	headerSivaFilenames:     "SIVA_FILENAMES",
	headerFileCount:         "FILE_COUNT",
	headerLangs:             "LANGS",
	headerLangsByteCount:    "LANGS_BYTE_COUNT",
	headerLangsLinesCount:   "LANGS_LINES_COUNT",
	headerLangsFilesCount:   "LANGS_FILES_COUNT",
	headerCommitsCount:      "COMMITS_COUNT",
	headerBranchesCount:     "BRANCHES_COUNT",
	headerForkCount:         "FORK_COUNT",
	headerEmptyLinesCount:   "EMPTY_LINES_COUNT",
	headerCodeLinesCount:    "CODE_LINES_COUNT",
	headerCommentLinesCount: "COMMENT_LINES_COUNT",
	headerLicense:           "LICENSE",
	headerStars:             "STARS",
	headerSize:              "SIZE",
}

// RepositoryFromCSV returns a repository given a CSV representation of it.
func RepositoryFromCSV(cols []string) (repo *Repository, err error) {
	p := parser{cols: cols}
	return &Repository{
		URL:                   p.string(headerURL),
		Filenames:             p.stringList(headerSivaFilenames),
		Files:                 p.int(headerFileCount),
		Languages:             p.stringList(headerLangs),
		LanguagesByteCount:    p.intList(headerLangsByteCount),
		LanguagesLineCount:    p.intList(headerLangsLinesCount),
		LanguagesFileCount:    p.intList(headerLangsFilesCount),
		Commits:               p.int(headerCommitsCount),
		Branches:              p.int(headerBranchesCount),
		Forks:                 p.int(headerForkCount),
		LanguagesEmptyLines:   p.intList(headerEmptyLinesCount),
		LanguagesCodeLines:    p.intList(headerCodeLinesCount),
		LanguagesCommentLines: p.intList(headerCommentLinesCount),
		License:               cols[headerLicense],
		Stars:                 p.int(headerStars),
		Size:                  p.int(headerSize),
	}, p.err
}

// ToCSV returns a slice of strings corresponding to the CSV representation of the repository.
func (r *Repository) ToCSV() []string {
	return []string{
		headerURL:               r.URL,
		headerSivaFilenames:     formatStringList(r.Filenames),
		headerFileCount:         formatInt(r.Files),
		headerLangs:             formatStringList(r.Languages),
		headerLangsByteCount:    formatIntList(r.LanguagesByteCount),
		headerLangsLinesCount:   formatIntList(r.LanguagesLineCount),
		headerLangsFilesCount:   formatIntList(r.LanguagesFileCount),
		headerCommitsCount:      formatInt(r.Commits),
		headerBranchesCount:     formatInt(r.Branches),
		headerForkCount:         formatInt(r.Forks),
		headerEmptyLinesCount:   formatIntList(r.LanguagesEmptyLines),
		headerCodeLinesCount:    formatIntList(r.LanguagesCodeLines),
		headerCommentLinesCount: formatIntList(r.LanguagesCommentLines),
		headerLicense:           r.License,
		headerStars:             formatInt(r.Stars),
		headerSize:              formatInt(r.Size),
	}
}

// Index provides iteration over repositories.
type Index interface {
	Next() (*Repository, error)
}

type csvIndex struct {
	r         *csv.Reader
	withStars bool
	withSize  bool
}

func (idx *csvIndex) Next() (*Repository, error) {
	rows, err := idx.r.Read()
	if err != nil {
		return nil, err
	}
	if rows[0] != csvHeaders[0] {
		if !idx.withStars {
			rows = append(rows, "-1")
		}

		if !idx.withSize {
			rows = append(rows, "-1")
		}

		return RepositoryFromCSV(rows)
	}
	return idx.Next()
}

var errBadHeader = fmt.Errorf("bad header, expected %s", strings.Join(csvHeaders, ","))

// IndexFromCSV returns an Index reading from a CSV file.
func IndexFromCSV(r io.Reader) (Index, error) {
	cr := csv.NewReader(r)
	headers, err := cr.Read()
	if err != nil {
		return nil, fmt.Errorf("could not skip headers row: %v", err)
	}

	// check for compatibility between old indexes
	length := len(headers)
	expected := len(csvHeaders)
	if length < expected-2 || length > expected {
		return nil, errBadHeader
	}

	var withStars, withSize bool
	for i := range headers {
		h := headers[i]
		if h != csvHeaders[i] {
			return nil, errBadHeader
		}

		switch h {
		case csvHeaders[headerStars]:
			withStars = true
		case csvHeaders[headerSize]:
			withSize = true
		}
	}

	return &csvIndex{cr, withStars, withSize}, nil
}

// A Filter provides a way to filter repositories.
type Filter func(*Repository) bool

type filterIndex struct {
	index  Index
	filter Filter
}

// Next returns the next element that matches all of the filters.
func (idx filterIndex) Next() (*Repository, error) {
	for {
		r, err := idx.index.Next()
		if err != nil {
			return nil, err
		}
		if idx.filter(r) {
			return r, nil
		}
	}
}

// WithFilter returns a new Index that only includes the repositories matching the given filter.
func WithFilter(index Index, filter Filter) Index {
	return &filterIndex{index, filter}
}

type parser struct {
	cols []string
	err  error
}

func (p *parser) string(idx int) string { return p.cols[idx] }

func (p *parser) stringList(idx int) []string {
	s := p.cols[idx]
	if s == "" {
		return nil
	}
	return strings.Split(p.cols[idx], ",")
}

func (p *parser) int(idx int) int64 {
	if p.err != nil {
		return 0
	}

	s := p.cols[idx]
	if s == "" {
		return 0
	}

	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		p.err = fmt.Errorf("parsing %s integer %q: %v", csvHeaders[idx], s, err)
	}
	return v
}

func (p *parser) intList(idx int) []int64 {
	if p.err != nil {
		return nil
	}

	ts := p.stringList(idx)
	vs := make([]int64, len(ts))
	for i, t := range ts {
		v, err := strconv.ParseInt(t, 10, 64)
		if err != nil {
			p.err = fmt.Errorf("could not parse %q in %s: %v", t, csvHeaders[idx], err)
			return nil
		}
		vs[i] = v
	}
	return vs
}

func formatStringList(l []string) string { return strings.Join(l, ",") }

func formatIntList(vs []int64) string {
	ts := make([]string, len(vs))
	for i, v := range vs {
		ts[i] = formatInt(v)
	}
	return formatStringList(ts)
}

func formatInt(v int64) string { return strconv.FormatInt(v, 10) }
