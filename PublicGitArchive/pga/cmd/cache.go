package cmd

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

const (
	indexURL  = "http://pga.sourced.tech/csv"
	indexName = "latest.csv.gz"
)

// copy checks whether a new version of the file in url exists and downloads it
// to dest. It returns true when the file exists, and an error when it was not possible
// to update it.
func copy(dest, source FileSystem, name string) (bool, error) {
	logrus.Debugf("syncing %s to %s", source.Abs(name), dest.Abs(name))
	localTime, err := dest.ModTime(name)
	exists := !os.IsNotExist(err)
	if err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("could not check mod time in %s: %v", dest.Abs(name), err)
	}

	remoteTime, err := source.ModTime(name)
	if err != nil {
		return exists, fmt.Errorf("could not check mod time in %s: %v", source.Abs(name), err)
	}

	if !localTime.IsZero() && !remoteTime.IsZero() && remoteTime.Before(localTime) {
		logrus.Debugf("local copy is up to date")
		return true, nil
	}
	logrus.Debugf("local copy is outdated or non existent")

	wc, err := dest.Create(name)
	if err != nil {
		return false, fmt.Errorf("could not create %s: %v", dest.Abs(name), err)
	}

	rc, err := source.Open(name)
	if err != nil {
		return false, err
	}
	defer rc.Close()

	if _, err = io.Copy(wc, rc); err != nil {
		return false, fmt.Errorf("could not copy %s to %s: %v", source.Abs(name), dest.Abs(name), err)
	}
	if err := wc.Close(); err != nil {
		return false, fmt.Errorf("could not close %s: %v", dest.Abs(name), err)
	}
	return true, nil
}

func getIndex() (io.ReadCloser, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, err
	}
	dest := localFS(filepath.Join(usr.HomeDir, ".pga"))
	source := urlFS(indexURL)

	ok, err := copy(dest, source, indexName)
	if !ok {
		return nil, err
	} else if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	f, err := dest.Open(indexName)
	if err != nil {
		return nil, err
	}
	return gzip.NewReader(f)
}
