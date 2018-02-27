package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/colinmarc/hdfs"
	"github.com/spf13/pflag"
)

type destination struct {
	path string
	fs   fs
}

func destinationFromFlags(flags *pflag.FlagSet) (*destination, error) {
	path, err := flags.GetString("output")
	if err != nil {
		return nil, err
	}

	if !strings.HasPrefix(path, "hdfs://") {
		return &destination{path, localDestination{}}, nil
	}

	client, err := hdfs.New(path)
	if err != nil {
		return nil, fmt.Errorf("could not create HDFS client: %v", err)
	}
	return &destination{path, hdfsDestination{client}}, nil
}

func (d destination) String() string { return fmt.Sprintf("%s %s", d.fs.name(), d.path) }

func (d destination) newWriter(name string) (io.WriteCloser, error) {
	dir := filepath.Join(d.path, "siva", "latest", name[:2])
	if err := os.MkdirAll(dir, 0777); err != nil {
		return nil, fmt.Errorf("could not create %s: %v", dir, err)
	}

	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("could not create %s: %v", path, err)
	}
	return f, nil
}

type fs interface {
	name() string

	MkdirAll(path string, perm os.FileMode) error
	Create(path string) (io.WriteCloser, error)
}

type localDestination struct{}

func (localDestination) name() string                                 { return "local destination" }
func (localDestination) MkdirAll(path string, perm os.FileMode) error { return os.MkdirAll(path, perm) }
func (localDestination) Create(path string) (io.WriteCloser, error)   { return os.Create(path) }

type hdfsDestination struct{ *hdfs.Client }

func (d hdfsDestination) name() string                               { return "HDFS destination" }
func (d hdfsDestination) Create(path string) (io.WriteCloser, error) { return d.Client.Create(path) }
