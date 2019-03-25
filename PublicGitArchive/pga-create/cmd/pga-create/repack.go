package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	humanize "github.com/dustin/go-humanize"
)

type repackCommand struct {
	URL    string `short:"l" long:"url" description:"Link to GHTorrent MySQL dump in tar.gz format. If empty (default), it find the most recent dump at GHTORRENT_MYSQL ?= http://ghtorrent-downloads.ewi.tudelft.nl/mysql/."`
	Stdin  bool   `long:"stdin" description:"read GHTorrent MySQL dump from stdin"`
	Output string `short:"o" long:"output" required:"true" description:"output file"`
}

func (c *repackCommand) Execute(args []string) error {
	repack(c.Stdin, c.URL, c.Output)

	return nil
}

func repack(stdin bool, url, output string) {
	startTime := time.Now()
	spin := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
	spin.Start()
	defer spin.Stop()

	inputFile := dumpReader(stdin, url, spin)
	var totalRead int64
	inputFile = trackingReader{RealReader: inputFile, Callback: func(n int) {
		totalRead += int64(n)
		if totalRead%3 == 0 {
			// supposing that mod 3 is random, this is a quick and dirty way to (/ 3) updates.
			spin.Suffix = fmt.Sprintf(" %s", humanize.Bytes(uint64(totalRead)))
		}
	}}
	gzf, err := gzip.NewReader(inputFile)
	if err != nil {
		fail("opening gzip stream", err)
	}
	defer gzf.Close()
	tarf := tar.NewReader(gzf)
	processed := int64(0)

	outf, err := createOutput(output)
	if err != nil {
		fail("creating output file", err)
	}
	gzw := gzip.NewWriter(outf)
	tarw := tar.NewWriter(gzw)

	numTasks := 3
	status := 0
	i := 0
	for header, err := tarf.Next(); err != io.EOF; header, err = tarf.Next() {
		if err != nil {
			fail("reading tar.gz", err)
		}
		i++
		processed += header.Size
		isRelevant := strings.HasSuffix(header.Name, "watchers.csv") ||
			strings.HasSuffix(header.Name, "projects.csv") ||
			strings.HasSuffix(header.Name, "project_languages.csv")
		mark := " "
		if isRelevant {
			mark = ">"
		}
		strSize := humanize.Bytes(uint64(header.Size))
		if strings.HasSuffix(strSize, " B") {
			strSize += " "
		}
		if i == 1 {
			fmt.Print("\r", strings.Repeat(" ", 80))
		}
		fmt.Printf("\r%s %2d  %7s  %s\n", mark, i, strSize, header.Name)
		if isRelevant {
			if err := tarw.WriteHeader(header); err != nil {
				fail("writing tar header", err)
			}

			if _, err := io.Copy(tarw, tarf); err != nil {
				fail("writing tar file", err)
			}

			status++
		}
		if status == numTasks {
			break
		}
	}

	if err := tarw.Close(); err != nil {
		fail("closing output tar file", err)
	}

	if err := gzw.Close(); err != nil {
		fail("closing output gz file", err)
	}

	fmt.Printf("\nRead      %s\nProcessed %s\nElapsed   %s\n",
		humanize.Bytes(uint64(totalRead)), humanize.Bytes(uint64(processed)), time.Since(startTime))

	if err := outf.Close(); err != nil {
		fail("closing output file", err)
	}
}

func createOutput(f string) (io.WriteCloser, error) {
	if f == "-" {
		return os.Stdout, nil
	}

	if err := os.MkdirAll(filepath.Dir(f), os.FileMode(0755)); err != nil {
		return nil, err
	}

	return os.Create(f)
}
