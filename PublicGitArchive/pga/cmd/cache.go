package cmd

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os/user"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

const indexURL = "http://pga.sourced.tech/csv"

// updateCache checks whether a new version of the file in url exists and downloads it
// to dest. It returns an error when it was not possible to update it.
func updateCache(ctx context.Context, dest, source FileSystem, name string) error {
	logrus.Debugf("syncing %s to %s", source.Abs(name), dest.Abs(name))
	if upToDate(dest, source, name) {
		logrus.Debugf("local copy is up to date")
		return nil
	}

	logrus.Debugf("local copy is outdated or non existent")
	tmpName := name + ".tmp"
	if err := copy(ctx, source, dest, name, tmpName); err != nil {
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

func copy(ctx context.Context, source, dest FileSystem,
	sourceName, destName string) (err error) {

	wc, err := dest.Create(destName)
	if err != nil {
		return fmt.Errorf("could not create %s: %v", dest.Abs(destName), err)
	}

	rc, err := source.Open(sourceName)
	if err != nil {
		_ = wc.Close()
		return err
	}

	if _, err = cancelableCopy(ctx, wc, rc); err != nil {
		_ = rc.Close()
		_ = wc.Close()
		return fmt.Errorf("could not copy %s to %s: %v",
			source.Abs(sourceName), dest.Abs(destName), err)
	}

	if err := rc.Close(); err != nil {
		_ = wc.Close()
		return err
	}

	return wc.Close()
}

const copyBufferSize = 512 * 1024

func cancelableCopy(ctx context.Context, dst io.Writer, src io.Reader) (int64, error) {
	var written int64
	for {
		select {
		case <-ctx.Done():
			return written, fmt.Errorf("download interrupted")
		default:
		}

		w, err := io.CopyN(dst, src, copyBufferSize)
		written += w
		if err == io.EOF {
			return written, nil
		}

		if err != nil {
			return written, err
		}
	}
}

func upToDate(dest, source FileSystem, name string) bool {
	if matchHash(dest, source, name) {
		return true
	}

	localTime, err := dest.ModTime(name)
	if err != nil {
		return false
	}

	remoteTime, err := source.ModTime(name)
	if err != nil {
		logrus.Warnf("could not check mod time in %s: %v", dest.Abs(name), err)
		return false
	}

	return !localTime.IsZero() && !remoteTime.IsZero() && remoteTime.Before(localTime)
}

func matchHash(dest, source FileSystem, name string) bool {
	localHash, err := dest.MD5(name)
	if err != nil {
		return false
	}
	remoteHash, err := source.MD5(name)
	if err != nil {
		return false
	}
	return localHash == remoteHash
}

func getIndex(ctx context.Context) (io.ReadCloser, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, err
	}
	dest := localFS(filepath.Join(usr.HomeDir, ".pga"))
	source := urlFS(indexURL)

	if err := updateCache(ctx, dest, source, indexName); err != nil {
		return nil, err
	}

	f, err := dest.Open(indexName)
	if err != nil {
		return nil, err
	}

	return gzip.NewReader(f)
}
