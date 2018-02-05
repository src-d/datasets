package repository

import (
	"io"
	"os"
	"path"

	"github.com/colinmarc/hdfs"
	"gopkg.in/src-d/go-billy.v4"
)

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

// Copier is in charge either to obtain a file from the Remote filesystem implementation,
// or send it from local.
type Copier interface {
	CopyFromRemote(src, dst string, localFs billy.Filesystem) error
	CopyToRemote(src, dst string, localFs billy.Filesystem) error
}

// NewLocalCopier returns a Copier using as a remote a Billy filesystem
func NewLocalCopier(fs billy.Filesystem, bucket int) Copier {
	return &LocalCopier{fs, bucket}
}

type LocalCopier struct {
	fs         billy.Filesystem
	bucketSize int
}

func (c *LocalCopier) CopyFromRemote(src, dst string, localFs billy.Filesystem) error {
	bSrc := addBucketName(src, c.bucketSize)
	return c.copyFile(c.fs, localFs, bSrc, dst)
}

func (c *LocalCopier) CopyToRemote(src, dst string, localFs billy.Filesystem) error {
	bDst := addBucketName(dst, c.bucketSize)
	return c.copyFile(localFs, c.fs, src, bDst)
}

func (c *LocalCopier) copyFile(fromFs, toFs billy.Filesystem, from, to string) (err error) {
	src, err := fromFs.Open(from)
	if os.IsNotExist(err) {
		return nil
	}

	if err != nil {
		return err
	}
	defer checkClose(src, &err)

	dst, err := toFs.OpenFile(to, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(0644))
	if err != nil {
		return err
	}
	defer checkClose(dst, &err)

	_, err = io.Copy(dst, src)
	return err
}

// NewHDFSCopier returns a copier using as a remote an HDFS cluster.
// URL is the hdfs connection URL and base is the base path to store all the files.
// tmpBase is the path to store all temporary .copy files while copying, which defaults
// to base if empty.
func NewHDFSCopier(URL, base, tmpBase string, bucket int) Copier {
	if tmpBase == "" {
		tmpBase = base
	}
	return &HDFSCopier{url: URL, base: base, tmpBase: tmpBase, bucketSize: bucket}
}

type HDFSCopier struct {
	url        string
	base       string
	tmpBase    string
	client     *hdfs.Client
	bucketSize int
}

// CopyFromRemote copies the file from HDFS to the provided billy Filesystem. If the file exists locally is overridden.
// If a writer is actually overriding the file on HDFS, we will able to read it, but a previous version of it.
func (c *HDFSCopier) CopyFromRemote(src, dst string, localFs billy.Filesystem) (err error) {
	if err := c.initializeClient(); err != nil {
		return err
	}

	bSrc := addBucketName(src, c.bucketSize)
	rf, err := c.client.Open(path.Join(c.base, bSrc))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer checkClose(rf, &err)

	lf, err := localFs.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(0644))
	if err != nil {
		return err
	}
	defer checkClose(lf, &err)

	_, err = io.Copy(lf, rf)
	return
}

func (c *HDFSCopier) deleteIfExists(file string) error {
	if _, err := c.client.Stat(file); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	return c.client.Remove(file)
}

// CopyToRemote copies from the provided billy Filesystem to HDFS. If the file exists on HDFS it will be overridden.
// If other writer is actually copying the same file to HDFS this method will throw an error because the WORM principle
// (Write Once Read Many).
func (c *HDFSCopier) CopyToRemote(src, dst string, localFs billy.Filesystem) (err error) {
	bDst := addBucketName(dst, c.bucketSize)
	p := path.Join(c.base, bDst)
	if err := c.initializeClient(); err != nil {
		return err
	}

	pCopy := path.Join(c.tmpBase, dst+".copy")

	lf, err := localFs.Open(src)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer checkClose(lf, &err)

	if err := c.client.MkdirAll(path.Dir(p), os.FileMode(0644)); err != nil {
		return err
	}

	if err := c.client.MkdirAll(path.Dir(pCopy), os.FileMode(0644)); err != nil {
		return err
	}

	// TODO to avoid this, we should implement a 'truncate' flag in 'client.Create' method
	err = c.deleteIfExists(pCopy)
	if err != nil {
		return err
	}

	rf, err := c.client.Create(pCopy)
	if err != nil {
		return err
	}

	// Delete temporary file in case the process is stopped while copying
	defer func() {
		rf.Close()
		c.deleteIfExists(pCopy)
	}()

	_, err = io.Copy(rf, lf)

	checkClose(rf, &err)

	if err != nil {
		c.client.Remove(pCopy)
		return err
	}

	err = c.client.Rename(pCopy, p)

	if err != nil {
		return err
	}

	return
}

func (c *HDFSCopier) initializeClient() (err error) {
	if c.client != nil {
		return
	}
	c.client, err = hdfs.New(c.url)

	return
}

func checkClose(c io.Closer, err *error) {
	if cerr := c.Close(); cerr != nil && *err == nil {
		*err = cerr
	}
}
