package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

func main() {
	file := flag.String("f", "result.csv", "csv file path with the results")
	output := flag.String("o", "result_forks.csv", "path to store the resultant csv file")
	flag.Parse()

	f, err := os.Open(*file)
	if os.IsNotExist(err) {
		logrus.WithField("file", *file).Fatal("file does not exist")
	} else if err != nil {
		logrus.WithField("file", *file).Fatal(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			logrus.WithField("file", *file).WithField("err", err).
				Warn("error closing file")
		}
	}()

	r := csv.NewReader(f)

	if err := os.Remove(*output); err != nil && !os.IsNotExist(err) {
		logrus.WithField("file", *output).Fatal(err)
	}

	fout, err := os.Create(*output)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"file": *output,
			"err":  err,
		}).Fatal("cannot create output file")
	}
	defer func() {
		if err := fout.Close(); err != nil {
			logrus.WithField("file", *output).WithField("err", err).
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
		return
	}

	header, records := records[0], records[1:]
	records = setForks(records)

	if err := w.WriteAll(append([][]string{header}, records...)); err != nil {
		logrus.WithField("err", err).Fatal("unable to write records")
	}

	logrus.Info("finished setting forks for repos")
}

func setForks(records [][]string) [][]string {
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

	return records
}

func toi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
