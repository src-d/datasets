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
	URL       string   // URL of the repository.
	Filenames []string // Siva filenames.
	FileCount int64    // Number of files in the repository.
	Languages []string // Languages found in the repository.
}

// RepositoryFromCSV returns a repository given a CSV representation of it.
func RepositoryFromCSV(cols []string) (*Repository, error) {
	fileCount, err := strconv.ParseInt(cols[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("could not parse file count in %s: %v", cols[2], err)
	}

	return &Repository{
		URL:       cols[0],
		Filenames: strings.Split(cols[1], ","),
		FileCount: fileCount,
		Languages: strings.Split(cols[3], ","),
	}, nil
}

// Index provides iteration over repositories.
type Index interface {
	Next() (*Repository, error)
}

type csvIndex struct{ r *csv.Reader }

func (idx *csvIndex) Next() (*Repository, error) {
	rows, err := idx.r.Read()
	if err != nil {
		return nil, err
	}
	if rows[0] != expectedHeader[0] {
		return RepositoryFromCSV(rows)
	}
	return idx.Next()
}

var expectedHeader = []string{
	"URL",
	"SIVA_FILENAMES",
	"FILE_COUNT",
	"LANGS",
	"LANGS_BYTE_COUNT",
	"LANGS_LINES_COUNT",
	"LANGS_FILES_COUNT",
	"COMMITS_COUNT",
	"BRANCHES_COUNT",
	"FORK_COUNT",
	"EMPTY_LINES_COUNT",
	"CODE_LINES_COUNT",
	"COMMENT_LINES_COUNT",
	"LICENSE",
}

var errBadHeader = fmt.Errorf("bad header, expected %v", expectedHeader)

// IndexFromCSV returns an Index reading from a CSV file.
func IndexFromCSV(r io.Reader) (Index, error) {
	cr := csv.NewReader(r)
	header, err := cr.Read()
	if err != nil {
		return nil, fmt.Errorf("could not skip headers row: %v", err)
	}

	if len(header) != len(expectedHeader) {
		return nil, errBadHeader
	}
	for i := range header {
		if header[i] != expectedHeader[i] {
			return nil, errBadHeader
		}
	}
	return &csvIndex{cr}, nil
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
