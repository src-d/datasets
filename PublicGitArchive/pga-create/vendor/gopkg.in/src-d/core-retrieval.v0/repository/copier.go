package repository

import (
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"context"

	"github.com/colinmarc/hdfs"
	"gopkg.in/src-d/go-billy.v4"
	errors "gopkg.in/src-d/go-errors.v0"
)

// Fs is a filesystem implementation that has some operations such as
// opening files, opening file writers, renaming, etc.
type Fs interface {
	// Open a file reader.
	Open(string) (io.ReadCloser, error)
	// WriteTo a file. It returns a file writer.
	WriteTo(string) (io.WriteCloser, error)
	// Rename atomically a file from one path to another.
	Rename(src, dst string) error
	// DeleteIfExists deletes a file only if it exists.
	DeleteIfExists(string) error
	// Base returns the base path of the filesystem to write files to.
	Base() string
	// TempDir returns the base path for temporary directories.
	TempDir() string
}

// Copier is in charge of copying files from a local filesystem to the remote
// one and vice-versa. It can optionally bucket files on the remote filesystem.
type Copier struct {
	remote     Fs
	local      *localFs
	bucketSize int
}

// NewCopier creates a new copier.
func NewCopier(local billy.Filesystem, remote Fs, bucketSize int) *Copier {
	return &Copier{remote: remote, local: &localFs{local}, bucketSize: bucketSize}
}

// Local returns the local filesystem of the copier.
func (c *Copier) Local() billy.Filesystem {
	return c.local.Filesystem
}

// CopyToRemote copies a file from the local filesystem to the remote one.
func (c *Copier) CopyToRemote(ctx context.Context, src, dst string) error {
	dst = filepath.Join(
		c.remote.Base(),
		addBucketName(dst, c.bucketSize),
	)
	dstCopy := dst + ".copy"

	lf, err := c.local.Open(filepath.Join(c.local.Base(), src))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	defer checkClose(lf, &err)

	w, err := c.remote.WriteTo(dstCopy)
	if err != nil {
		return err
	}

	if err = copy(ctx, w, lf); err != nil {
		_ = w.Close()
		_ = c.remote.DeleteIfExists(dstCopy)
		return err
	}

	if err = w.Close(); err != nil {
		_ = c.remote.DeleteIfExists(dstCopy)
		return err
	}

	err = c.remote.Rename(dstCopy, dst)
	if err != nil {
		return err
	}

	return err
}

// CopyFromRemote copies a file to the local filesystem from the remote one.
func (c *Copier) CopyFromRemote(ctx context.Context, src, dst string) (err error) {
	src = filepath.Join(
		c.remote.Base(),
		addBucketName(src, c.bucketSize),
	)
	rf, err := c.remote.Open(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer checkClose(rf, &err)

	w, err := c.local.WriteTo(filepath.Join(c.local.Base(), dst))
	if err != nil {
		return err
	}
	defer checkClose(w, &err)

	err = copy(ctx, w, rf)
	return
}

type localFs struct {
	billy.Filesystem
}

// NewLocalFs returns a local file system.
func NewLocalFs(fs billy.Filesystem) Fs {
	return &localFs{fs}
}

func (fs *localFs) Open(path string) (io.ReadCloser, error) {
	return fs.Filesystem.Open(path)
}

func (fs *localFs) WriteTo(path string) (io.WriteCloser, error) {
	return fs.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(0644))
}

func (fs *localFs) DeleteIfExists(path string) error {
	if _, err := fs.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return err
	}

	return fs.Remove(path)
}

func (fs *localFs) TempDir() string {
	return os.TempDir()
}

func (fs *localFs) Base() string {
	return "/"
}

type hdfsFs struct {
	url     string
	base    string
	tmpBase string
	client  *hdfs.Client
}

// NewHDFSFs returns a filesystem that can access a HDFS cluster.
// URL is the hdfs connection URL and base is the base path to store all the files.
// tmpBase is the path to store all temporary .copy files while copying, which defaults
// to base if empty.
func NewHDFSFs(URL, base, tmpBase string) Fs {
	if tmpBase == "" {
		tmpBase = base
	}
	return &hdfsFs{url: URL, base: base, tmpBase: tmpBase}
}

// HDFSNamenodeError is returned when there is a namenode error.
var HDFSNamenodeError = errors.NewKind("HDFS namenode error")
var errorsToCatchFromHDFS = []string{
	"no available namenodes",
	"org.apache.hadoop.hdfs.server.namenode.SafeModeException",
}

func (fs *hdfsFs) freeClient(err error) (wrapped error) {
	wrapped = err
	if err != nil {
		for _, HDFSError := range errorsToCatchFromHDFS {
			if strings.Contains(err.Error(), HDFSError) {
				wrapped = HDFSNamenodeError.Wrap(err)
				fs.client = nil
				break
			}
		}
	}

	return
}

func (fs *hdfsFs) initializeClient() (err error) {
	if fs.client != nil {
		return
	}

	fs.client, err = hdfs.New(fs.url)
	return
}

func (fs *hdfsFs) Open(path string) (r io.ReadCloser, err error) {
	defer func() {
		err = fs.freeClient(err)
	}()

	if err = fs.initializeClient(); err != nil {
		return
	}

	r, err = fs.client.Open(path)
	if err != nil {
		return
	}

	return
}

func (fs *hdfsFs) WriteTo(path string) (w io.WriteCloser, err error) {
	defer func() {
		err = fs.freeClient(err)
	}()

	if err = fs.initializeClient(); err != nil {
		return
	}

	if err = fs.client.MkdirAll(filepath.Dir(path), os.FileMode(0644)); err != nil {
		return
	}

	if err = fs.DeleteIfExists(path); err != nil {
		return
	}

	w, err = fs.client.Create(path)
	if err != nil {
		return
	}

	return
}

func (fs *hdfsFs) Rename(src, dst string) (err error) {
	defer func() {
		err = fs.freeClient(err)
	}()

	if err = fs.client.MkdirAll(filepath.Dir(dst), os.FileMode(0644)); err != nil {
		return err
	}

	err = fs.client.Rename(src, dst)
	return
}

func (fs *hdfsFs) DeleteIfExists(file string) (err error) {
	defer func() {
		err = fs.freeClient(err)
	}()

	if _, err = fs.client.Stat(file); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	if err = fs.client.Remove(file); err != nil {
		return err
	}

	return
}

func (fs *hdfsFs) TempDir() string {
	return fs.tmpBase
}

func (fs *hdfsFs) Base() string {
	return fs.base
}

func checkClose(c io.Closer, err *error) {
	if cerr := c.Close(); cerr != nil && *err == nil {
		*err = cerr
	}
}

const copySize = 64 * 1024

// ErrCopyCancelled is returned when a copy is cancelled.
var ErrCopyCancelled = errors.NewKind("copy was cancelled")

func copy(ctx context.Context, dst io.Writer, src io.Reader) error {
	buf := make([]byte, copySize)
	var done bool

	for {
		select {
		case <-ctx.Done():
			return ErrCopyCancelled.New()
		default:
		}

		n, err := src.Read(buf)
		if err != nil {
			if err != io.EOF {
				return err
			}

			done = true
		}

		if n > 0 {
			nw, err := dst.Write(buf[0:n])
			if err != nil {
				return err
			}

			if n != nw {
				return io.ErrShortWrite
			}
		}

		if done {
			return nil
		}
	}
}

// addBucketName prepends the bucket dir name to the file name. The number of
// characters used for the directory are specified by bucketSize. A value of 0
// in bucketSize returns the name as is.
func addBucketName(name string, bucketSize int) string {
	if bucketSize > 0 {
		bucket := string(name[0:bucketSize])
		newPath := path.Join(bucket, name)

		return newPath
	}

	return name
}
