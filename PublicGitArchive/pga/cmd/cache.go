package cmd

import (
	"compress/gzip"
	"fmt"
	"io"
	"os/user"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

const (
	indexURL  = "http://pga.sourced.tech/csv"
	indexName = "latest.csv.gz"
)

// updateCache checks whether a new version of the file in url exists and downloads it
// to dest. It returns an error when it was not possible to update it.
func updateCache(dest, source FileSystem, name string) error {
	logrus.Debugf("syncing %s to %s", source.Abs(name), dest.Abs(name))
	if upToDate(dest, source, name) {
		logrus.Debugf("local copy is up to date")
		return nil
	}

	logrus.Debugf("local copy is outdated or non existent")
	wc, err := dest.Create(name)
	if err != nil {
		return fmt.Errorf("could not create %s: %v", dest.Abs(name), err)
	}

	rc, err := source.Open(name)
	if err != nil {
		return err
	}
	defer rc.Close()

	if _, err = io.Copy(wc, rc); err != nil {
		return fmt.Errorf("could not copy %s to %s: %v", source.Abs(name), dest.Abs(name), err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("could not close %s: %v", dest.Abs(name), err)
	}
	return nil
}

func upToDate(dest, source FileSystem, name string) bool {
	ok, err := matchHash(dest, source, name)
	if err == nil {
		return ok
	}
	logrus.Warnf("could not check md5 hashes for %s, comparing timestamps instead: %v", name, err)

	localTime, err := dest.ModTime(name)
	if err != nil {
		logrus.Warnf("could not check mod time in %s: %v", dest.Abs(name), err)
		return false
	}

	remoteTime, err := source.ModTime(name)
	if err != nil {
		logrus.Warnf("could not check mod time in %s: %v", dest.Abs(name), err)
		return false
	}

	return !localTime.IsZero() && !remoteTime.IsZero() && remoteTime.Before(localTime)
}

func matchHash(dest, source FileSystem, name string) (bool, error) {
	localHash, err := dest.MD5(name)
	if err != nil {
		return false, err
	}
	remoteHash, err := source.MD5(name)
	if err != nil {
		return false, err
	}
	return localHash == remoteHash, nil
}

func getIndex() (io.ReadCloser, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, err
	}
	dest := localFS(filepath.Join(usr.HomeDir, ".pga"))
	source := urlFS(indexURL)

	if err := updateCache(dest, source, indexName); err != nil {
		return nil, err
	}

	f, err := dest.Open(indexName)
	if err != nil {
		return nil, err
	}
	return gzip.NewReader(f)
}
