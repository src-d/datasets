package main

import (
	"compress/gzip"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
)

type selectCommand struct {
	Repositories string `short:"r" long:"repositories" default:"data/repositories.csv.gz" description:"Input path for the gzipped file with the repositories extracted information."`
	IndexList    string `short:"i" long:"index-list" default:"data/repositories-index.csv.gz" description:"Output path for the list that should be given to the index command in case min stars filter applies."`
	MinStars     int    `short:"m" long:"min-stars" default:"0" description:"Minimum number of stars. <1 means no filter"`
	Max          int    `short:"n" long:"max" default:"-1" description:"Maximum number of top-starred repositories to clone. -1 means unlimited."`
	URLTemplate  string `long:"url-template" default:"git://github.com/%s.git" description:"Output URL printf template."`
}

func (c *selectCommand) Execute(args []string) error {
	selectRepos(selectionParameters{
		ReposFile:   c.Repositories,
		IndexList:   c.IndexList,
		MinStars:    c.MinStars,
		TopN:        c.Max,
		URLTemplate: c.URLTemplate,
	})

	return nil
}

type selectionParameters struct {
	ReposFile   string
	IndexList   string
	MinStars    int
	TopN        int
	URLTemplate string
}

func selectRepos(params selectionParameters) {
	f, err := os.Open(params.ReposFile)
	if err != nil {
		fail("opening repositories file "+params.ReposFile, err)
	}
	defer f.Close()

	gzf, err := gzip.NewReader(f)
	if err != nil {
		fail("decompressing repositories file "+params.ReposFile, err)
	}
	defer gzf.Close()

	var idxw *csv.Writer
	if params.MinStars > 0 {
		idxList, err := os.Create(params.IndexList)
		if err != nil {
			fail("creating file "+params.IndexList, err)
		}
		defer func() {
			if err := idxList.Close(); err != nil {
				fail("closing output file "+params.IndexList, err)
			}
		}()

		gzw := gzip.NewWriter(idxList)
		defer func() {
			if err := gzw.Close(); err != nil {
				fail("closing output gz file "+params.IndexList, err)
			}
		}()

		idxw = csv.NewWriter(gzw)
	}

	r := csv.NewReader(gzf)
	// read headers
	_, err = r.Read()
	if err != nil {
		fail("parsing repositories file "+params.ReposFile, err)
	}

	var count int
	for record, err := r.Read(); err != io.EOF; record, err = r.Read() {
		if err != nil {
			fail("parsing repositories file "+params.ReposFile, err)
		}

		if params.TopN > -1 && count >= params.TopN {
			break
		}

		if params.MinStars > 0 {
			stars, err := strconv.Atoi(record[1])
			if err != nil {
				fail("parsing stars field from repositories file "+params.ReposFile, err)
			}

			if stars < params.MinStars {
				// the file is sorted
				break
			}

			if err := idxw.Write(record); err != nil {
				fail("writing repository to index list file", err)
			}
		}

		fmt.Fprintf(os.Stdout, params.URLTemplate+"\n", record[0])
		count++
	}

	if params.MinStars > 0 {
		idxw.Flush()
		if err := idxw.Error(); err != nil {
			fail("writing repositories to index list file", err)
		}
	}
}
