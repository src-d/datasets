package main

import (
	"io/ioutil"
	"os"
	"runtime"
	"strings"

	"github.com/src-d/datasets/PublicGitArchive/pga-create"

	"github.com/sirupsen/logrus"
	"gopkg.in/src-d/core-retrieval.v0"
)

type indexCommand struct {
	Output    string `short:"o" long:"output" default:"result.csv" description:"csv file path with the results"`
	Debug     bool   `long:"debug" description:"show debug logs"`
	LogFile   string `long:"logfile" description:"write logs to file"`
	Limit     uint64 `long:"limit" description:"max number of repositories to process"`
	Offset    uint64 `long:"offset" description:"skip initial n repositories"`
	Workers   int    `long:"workers" description:"number of workers to use (defaults to number of CPUs)"`
	ReposFile string `long:"repos-file" description:"path to a file with a repository per line, only those will be processed"`
}

func (c *indexCommand) Execute(args []string) error {
	if c.Workers <= 0 {
		c.Workers = runtime.NumCPU()
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

	var repos []string
	if c.ReposFile != "" {
		content, err := ioutil.ReadFile(c.ReposFile)
		if err != nil {
			logrus.WithField("err", err).Fatalf("unable to read repositories file: %s", c.ReposFile)
		}

		for _, r := range strings.Split(string(content), "\n") {
			if strings.TrimSpace(r) != "" {
				repos = append(repos, r)
			}
		}
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
