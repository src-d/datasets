package cmd

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/colinmarc/hdfs"
	"github.com/spf13/pflag"
)

// FileSystem provides an abstraction over various file systems.
type FileSystem interface {
	Abs(path string) string
	Create(path string) (io.WriteCloser, error)
	Open(path string) (io.ReadCloser, error)
	ModTime(path string) (time.Time, error)
	Size(path string) (int64, error)
	MD5(path string) (string, error)
}

// FileSystemFromFlags returns the correct file system given a set of flags.
func FileSystemFromFlags(flags *pflag.FlagSet) (FileSystem, error) {
	path, err := flags.GetString("output")
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("could not parse output location: %v", err)
	}

	switch u.Scheme {
	case "http", "https":
		return urlFS(path), nil
	case "hdfs":
		client, err := hdfs.New(u.Host)
		if err != nil {
			return nil, fmt.Errorf("could not create HDFS client: %v", err)
		}
		return hdfsFS{u.Path, client}, nil
	case "":
		return localFS(path), nil
	default:
		return nil, fmt.Errorf("scheme not supported in output location %s", path)
	}
}

type localFS string

func (fs localFS) Abs(path string) string                  { return filepath.Join(string(fs), path) }
func (fs localFS) Open(path string) (io.ReadCloser, error) { return os.Open(fs.Abs(path)) }
func (fs localFS) ModTime(path string) (time.Time, error)  { return modtime(os.Stat(fs.Abs(path))) }
func (fs localFS) Size(path string) (int64, error)         { return size(os.Stat(fs.Abs(path))) }
func (fs localFS) MD5(path string) (string, error)         { return md5Hash(fs, path) }

func md5Hash(fs FileSystem, path string) (string, error) {
	rc, err := fs.Open(path)
	if err != nil {
		return "", err
	}
	defer rc.Close()

	w := md5.New()
	if _, err := io.Copy(w, rc); err != nil {
		return "", fmt.Errorf("could not copy to hash: %v", err)
	}
	return hex.EncodeToString(w.Sum(nil)), nil
}

func (fs localFS) Create(path string) (io.WriteCloser, error) {
	path = fs.Abs(path)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf("could not create %s: %v", dir, err)
	}
	return os.Create(path)
}

type urlFS string

func (fs urlFS) Abs(path string) string { return string(fs) + "/" + path }

func (fs urlFS) Create(path string) (io.WriteCloser, error) {
	return nil, fmt.Errorf("not implemented for URLs")
}

func (fs urlFS) Open(path string) (io.ReadCloser, error) {
	res, err := http.Get(fs.Abs(path))
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(res.Status)
	}
	return res.Body, nil
}

func (fs urlFS) ModTime(path string) (time.Time, error) {
	h, err := fs.header(path)
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse("Mon, 02 Jan 2006 15:04:05 MST", h.Get("Last-Modified"))
}

func (fs urlFS) Size(path string) (int64, error) {
	h, err := fs.header(path)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(h.Get("Content-Length"), 10, 64)
}

func (fs urlFS) header(path string) (http.Header, error) {
	req, _ := http.NewRequest(http.MethodHead, fs.Abs(path), nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(res.Status)
	}
	return res.Header, nil
}

func (fs urlFS) MD5(path string) (string, error) {
	url := fs.Abs(path) + ".md5"
	res, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("could not fetch hash at %s.md5: %v", path, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("could not fetch hash at %s.md5: %s", path, res.Status)
	}
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("could not read md5 hash: %v", err)
	}
	return string(bytes.Fields(b)[0]), nil
}

type hdfsFS struct {
	path string
	c    *hdfs.Client
}

func (fs hdfsFS) Abs(path string) string { return fs.path + "/" + path }

func (fs hdfsFS) Create(path string) (io.WriteCloser, error) {
	path = fs.Abs(path)
	dir := filepath.Dir(path)
	if err := fs.c.MkdirAll(dir, os.ModePerm); err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf("could not create %s: %v", dir, err)
	}
	return fs.c.Create(path)
}

func (fs hdfsFS) Open(path string) (io.ReadCloser, error) { return fs.c.Open(fs.Abs(path)) }
func (fs hdfsFS) ModTime(path string) (time.Time, error)  { return modtime(fs.c.Stat(fs.Abs(path))) }
func (fs hdfsFS) Size(path string) (int64, error)         { return size(fs.c.Stat(fs.Abs(path))) }
func (fs hdfsFS) MD5(path string) (string, error)         { return md5Hash(fs, path) }

func modtime(fi os.FileInfo, err error) (time.Time, error) {
	if err != nil {
		return time.Time{}, err
	}
	return fi.ModTime(), nil
}

func size(fi os.FileInfo, err error) (int64, error) {
	if err != nil {
		return 0, err
	}
	return fi.Size(), nil
}
