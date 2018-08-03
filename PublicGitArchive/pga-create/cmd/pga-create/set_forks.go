package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

type setForksCommand struct {
	File   string `short:"f" long:"file" default:"result.csv" description:"csv file path with the results"`
	Output string `short:"o" long:"output" default:"result_forks.csv" description:"path to store the resultant csv file"`
}

func (c *setForksCommand) Execute(args []string) error {
	f, err := os.Open(c.File)
	if os.IsNotExist(err) {
		logrus.WithField("file", c.File).Fatal("file does not exist")
	} else if err != nil {
		logrus.WithField("file", c.File).Fatal(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			logrus.WithField("file", c.File).WithField("err", err).
				Warn("error closing file")
		}
	}()

	r := csv.NewReader(f)

	if err := os.Remove(c.Output); err != nil && !os.IsNotExist(err) {
		logrus.WithField("file", c.Output).Fatal(err)
	}

	fout, err := os.Create(c.Output)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"file": c.Output,
			"err":  err,
		}).Fatal("cannot create output file")
	}
	defer func() {
		if err := fout.Close(); err != nil {
			logrus.WithField("file", c.Output).WithField("err", err).
				Warn("error closing file")
		}
	}()

	w := csv.NewWriter(fout)
	defer w.Flush()

	records, err := r.ReadAll()
	if err != nil {
		logrus.WithField("err", err).Fatal("unable to read csv file")
	}

	if len(records) == 0 {
		logrus.Warn("empty csv file")
		return nil
	}

	header, records := records[0], records[1:]
	resetForks(records)
	setForks(records)

	if err := w.WriteAll(append([][]string{header}, records...)); err != nil {
		logrus.WithField("err", err).Fatal("unable to write records")
	}

	logrus.Info("finished setting forks for repos")
	return nil
}

func setForks(records [][]string) {
	var reposBySiva = make(map[string][]string)
	for _, r := range records {
		for _, s := range strings.Split(r[ /*SIVA_FILENAMES*/ 1], ",") {
			reposBySiva[s] = append(reposBySiva[s], r[ /*URL*/ 0])
		}
	}

	for _, r := range records {
		for _, s := range strings.Split(r[ /*SIVA_FILENAMES*/ 1], ",") {
			r[ /*FORK_COUNT*/ 9] = fmt.Sprint(toi(r[9]) + len(reposBySiva[s]) - 1)
		}
	}
}

func resetForks(records [][]string) {
	for _, r := range records {
		r[ /* FORK_COUNT*/ 9] = "0"
	}
}

func toi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
