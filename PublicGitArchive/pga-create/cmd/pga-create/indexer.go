package main

import (
	"compress/gzip"
	"encoding/csv"
	"io"
	"os"
	"runtime"
	"strconv"

	indexer "github.com/src-d/datasets/PublicGitArchive/pga-create"

	"github.com/sirupsen/logrus"
	"gopkg.in/src-d/core-retrieval.v0"
)

type indexCommand struct {
	Output       string `short:"o" long:"output" default:"data/index.csv" description:"csv file path with the results"`
	Debug        bool   `long:"debug" description:"show debug logs"`
	LogFile      string `long:"logfile" description:"write logs to file"`
	Limit        uint64 `long:"limit" description:"max number of repositories to process"`
	Offset       uint64 `long:"offset" description:"skip initial n repositories"`
	Workers      int    `long:"workers" description:"number of workers to use (defaults to GOMAXPROCS)"`
	Repositories string `short:"r" long:"repositories" default:"data/repositories.csv.gz" description:"input path for the gzipped file with the repositories extracted information"`
}

func (c *indexCommand) Execute(args []string) error {
	if c.Workers <= 0 {
		c.Workers = runtime.GOMAXPROCS(-1)
	}

	if c.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}

	if c.LogFile != "" {
		_ = os.Remove(c.LogFile)
		f, err := os.Create(c.LogFile)
		if err != nil {
			logrus.WithField("err", err).Fatalf("unable to create log file: %s", c.LogFile)
		}

		defer func() {
			if err := f.Close(); err != nil {
				logrus.WithField("err", err).Error("unable to close log file")
			}
		}()

		logrus.SetOutput(f)
	}

	repos, err := loadReposList(c.Repositories)
	if err != nil {
		logrus.WithField("err", err).Fatal("unable to read " + c.Repositories)
	}

	indexer.Index(
		core.ModelRepositoryStore(),
		core.RootedTransactioner(),
		c.Output,
		c.Workers,
		c.Limit,
		c.Offset,
		repos,
	)

	return nil
}

func loadReposList(path string) (map[string]uint32, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gzr.Close()

	r := csv.NewReader(gzr)
	// read headers
	if _, err := r.Read(); err != nil {
		return nil, err
	}

	repos := map[string]uint32{}
	for record, err := r.Read(); err != io.EOF; record, err = r.Read() {
		stars, err := strconv.Atoi(record[1])
		if err != nil {
			return nil, err
		}

		repo := record[0]
		repos[repo] = uint32(stars)
	}

	return repos, nil
}
