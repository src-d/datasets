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
	tmpName := name + ".tmp"
	if err := copy(source, dest, name, tmpName); err != nil {
		logrus.Warningf("copy from % to %s failed: %v",
			source.Abs(name), dest.Abs(tmpName), err)
		if cerr := dest.Remove(tmpName); cerr != nil {
			logrus.Warningf("error removing temporary file %s: %v",
				dest.Abs(tmpName), cerr)
		}

		return fmt.Errorf("could not copy to temporary file %s: %v",
			dest.Abs(tmpName), err)
	}

	if err := dest.Rename(tmpName, name); err != nil {
		return fmt.Errorf("rename %s to %s failed: %v",
			dest.Abs(tmpName), dest.Abs(name), err)
	}

	return nil
}

func copy(source, dest FileSystem, sourceName, destName string) (err error) {
	wc, err := dest.Create(destName)
	if err != nil {
		return fmt.Errorf("could not create %s: %v", dest.Abs(destName), err)
	}
	defer checkClose(dest.Abs(destName), wc, &err)

	rc, err := source.Open(sourceName)
	if err != nil {
		return err
	}
	defer checkClose(source.Abs(sourceName), rc, &err)

	if _, err = io.Copy(wc, rc); err != nil {
		return fmt.Errorf("could not copy %s to %s: %v",
			source.Abs(sourceName), dest.Abs(destName), err)
	}

	return err
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

func checkClose(name string, c io.Closer, err *error) {
	if cerr := c.Close(); cerr != nil && *err == nil {
		*err = fmt.Errorf("could not close %s: %v", name, cerr)
	}
}
