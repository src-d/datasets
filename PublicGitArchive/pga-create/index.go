package indexer

import (
	"bufio"
	"compress/gzip"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/src-d/core-retrieval.v0/model"
	"gopkg.in/src-d/core-retrieval.v0/repository"
	"gopkg.in/src-d/go-kallax.v1"
)

// Index to the given output csv file all the processed repositories in
// the given store.
func Index(
	store *model.RepositoryStore,
	txer repository.RootedTransactioner,
	outputFile string,
	workers int,
	limit uint64,
	offset uint64,
	list []string,
	reposIDPath string,
	starsPath string,
) {
	f, err := createOutputFile(outputFile)
	if err != nil {
		logrus.WithField("file", outputFile).WithField("err", err).
			Fatal("unable to create file")
	}
	defer f.Close()

	w := csv.NewWriter(f)
	if err := w.Write(csvHeader); err != nil {
		logrus.WithField("err", err).Fatal("unable to write csv header")
	}
	w.Flush()

	rs, total, err := getResultSet(store, limit, offset, list)
	if err != nil {
		logrus.WithField("err", err).Fatal("unable to get result set")
	}

	stars, err := getRepoToStars(reposIDPath, starsPath, list)
	if err != nil {
		logrus.WithField("err", err).Fatal("unable to get repositories' stars")
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	repos := processRepos(workers, txer, rs, stars)
	var processed int
	for {
		select {
		case repo, ok := <-repos:
			if !ok {
				logrus.WithFields(logrus.Fields{
					"processed": processed,
					"failed":    total - int64(processed),
					"total":     total,
				}).Info("finished processing all repositories")
				return
			}

			logrus.WithField("repo", repo.URL).Debug("writing record to CSV")
			if err := w.Write(repo.toRecord()); err != nil {
				logrus.WithFields(logrus.Fields{
					"err":  err,
					"repo": repo.URL,
				}).Fatal("unable to write csv record")
			}
			w.Flush()
			processed++
		case <-signals:
			logrus.Warn("received an interrupt signal, stopping")
			return
		}
	}
}

func createOutputFile(outputFile string) (*os.File, error) {
	if _, err := os.Stat(outputFile); err != nil && !os.IsNotExist(err) {
		return nil, err
	} else if err == nil {
		logrus.WithField("file", outputFile).Warn("file exists, it will be deleted")
		if err := os.Remove(outputFile); err != nil {
			return nil, err
		}
	}

	return os.Create(outputFile)
}

func getResultSet(
	store *model.RepositoryStore,
	limit, offset uint64,
	list []string,
) (*model.RepositoryResultSet, int64, error) {
	query := model.NewRepositoryQuery().
		FindByStatus(model.Fetched).
		WithReferences(nil)

	var repos = make([]interface{}, len(list))
	for i, r := range list {
		repos[i] = r
	}

	if len(list) > 0 {
		query = query.Where(kallax.ArrayOverlap(
			model.Schema.Repository.Endpoints,
			repos...,
		))
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	if offset > 0 {
		query = query.Offset(offset)
	}

	total, err := store.Count(query)
	if err != nil {
		return nil, 0, err
	}

	rs, err := store.Find(query.Order(kallax.Asc(model.Schema.Repository.ID)))
	if err != nil {
		return nil, 0, err
	}

	return rs, total, nil
}

func getRepoToStars(reposIDPath, starsPath string, list []string) (map[string]int, error) {
	r, err := os.Open(reposIDPath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	s, err := os.Open(starsPath)
	if err != nil {
		return nil, err
	}
	defer s.Close()

	rgz, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	defer rgz.Close()

	sgz, err := gzip.NewReader(s)
	if err != nil {
		return nil, err
	}
	defer sgz.Close()

	var repoSet map[string]struct{}
	if len(list) != 0 {
		repoSet = reposListToSet(list)
	}

	repos, err := buildIDToRepo(rgz, repoSet)
	if err != nil {
		return nil, err
	}

	var idSet map[int]struct{}
	if len(list) != 0 {
		idSet = make(map[int]struct{}, len(repos))
		for id := range repos {
			idSet[id] = struct{}{}
		}
	}

	stars, err := buildIDToStars(sgz, idSet)
	if err != nil {
		return nil, err
	}

	repoStars := make(map[string]int)
	for id, repo := range repos {
		// if id is not present in stars map that repo has no stars.
		n, ok := stars[id]
		if ok {
			repoStars[repo] = n
		}
	}

	return repoStars, nil
}

func reposListToSet(list []string) map[string]struct{} {
	if len(list) == 0 {
		return nil
	}

	repos := make(map[string]struct{}, len(list))
	for _, url := range list {
		name := trimRepoURL(url)
		repos[name] = struct{}{}
	}

	return repos
}

func trimRepoURL(url string) string {
	const (
		HTTPprefix = "https://github.com/"
		SSHprefix  = "git://github.com/"
		suffix     = ".git"
	)

	var repo string
	if strings.HasPrefix(url, HTTPprefix) {
		repo = strings.TrimPrefix(url, HTTPprefix)
	} else if strings.HasPrefix(url, SSHprefix) {
		repo = strings.TrimPrefix(url, SSHprefix)
		repo = strings.TrimSuffix(repo, suffix)
	}

	return repo
}

func buildIDToRepo(r io.Reader, repoSet map[string]struct{}) (map[int]string, error) {
	repos := make(map[int]string)
	scanner := bufio.NewScanner(r)
	var count int
	for scanner.Scan() {
		var (
			id   int
			name string
		)

		line := scanner.Text()
		if line == "" {
			continue
		}

		if _, err := fmt.Sscan(line, &id, &name); err != nil {
			return nil, err
		}

		_, ok := repoSet[name]
		if len(repoSet) == 0 || ok {
			repos[id] = name
			count++
		}

		if len(repoSet) != 0 && count >= len(repoSet) {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return repos, nil
}

func buildIDToStars(r io.Reader, idSet map[int]struct{}) (map[int]int, error) {
	stars := make(map[int]int)
	scanner := bufio.NewScanner(r)
	var count int
	for scanner.Scan() {
		var id, nstar int

		line := scanner.Text()
		if line == "" {
			continue
		}

		if _, err := fmt.Sscan(line, &id, &nstar); err != nil {
			return nil, err
		}

		_, ok := idSet[id]
		if len(idSet) == 0 || ok {
			stars[id] = nstar
			count++
		}

		if len(idSet) != 0 && count >= len(idSet) {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return stars, nil
}
