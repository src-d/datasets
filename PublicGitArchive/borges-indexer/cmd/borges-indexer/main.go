package main

import (
	"flag"
	"io/ioutil"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/src-d/datasets/PublicGitArchive/borges-indexer"
	"gopkg.in/src-d/core-retrieval.v0"
)

func main() {
	output := flag.String("o", "result.csv", "csv file path with the results")
	debug := flag.Bool("debug", false, "show debug logs")
	logfile := flag.String("logfile", "", "write logs to file")
	limit := flag.Uint64("limit", 0, "max number of repositories to process")
	offset := flag.Uint64("offset", 0, "skip initial n repositories")
	reposFile := flag.String("repos-file", "", "path to a file with a repository per line, only those will be processed")
	flag.Parse()

	if *debug {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}

	if *logfile != "" {
		_ = os.Remove(*logfile)
		f, err := os.Create(*logfile)
		if err != nil {
			logrus.WithField("err", err).Fatalf("unable to create log file: %s", *logfile)
		}

		defer func() {
			if err := f.Close(); err != nil {
				logrus.WithField("err", err).Error("unable to close log file")
			}
		}()

		logrus.SetOutput(f)
	}

	var repos []string
	if *reposFile != "" {
		content, err := ioutil.ReadFile(*reposFile)
		if err != nil {
			logrus.WithField("err", err).Fatalf("unable to read repositories file: %s", *reposFile)
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
		*output,
		*limit,
		*offset,
		repos,
	)
}
