package cmd

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"time"
)

var (
	indexURL  = "http://pga.sourced.tech/csv/latest.csv.gz"
	cacheDir  = ".pga"
	cachePath = "cache.cvs.gz"
)

func getIndex() (io.ReadCloser, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, err
	}
	dest := filepath.Join(usr.HomeDir, cacheDir, cachePath)

	var cacheModTime, indexModTime time.Time

	fi, err := os.Stat(dest)
	if err == nil {
		cacheModTime = fi.ModTime()
	}

	req, _ := http.NewRequest(http.MethodHead, indexURL, nil)
	res, err := http.DefaultClient.Do(req)
	if err == nil && res.StatusCode == http.StatusOK {
		indexModTime, _ = time.Parse("Mon, 02 Jan 2006 15:04:05 MST", res.Header.Get("Last-Modified"))
	}

	if cacheModTime.IsZero() || cacheModTime.Before(indexModTime) {
		if err := refreshCache(dest); err != nil {
			return nil, err
		}
	}

	f, err := os.Open(dest)
	if err != nil {
		return nil, err
	}
	return gzip.NewReader(f)
}

func refreshCache(dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0777); err != nil && err != os.ErrExist {
		return fmt.Errorf("could not create %s: %v", filepath.Dir(dest), err)
	}

	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("could not create %s: %v", cachePath, err)
	}
	fmt.Fprintf(os.Stderr, "INFO: storing index cache in %s\n", dest)
	res, err := http.Get(indexURL)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("status: %s", res.Status)
	}

	_, err = io.Copy(f, res.Body)
	if err != nil {
		return fmt.Errorf("could not copy file: %v", err)
	}
	return nil
}
