package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	humanize "github.com/dustin/go-humanize"
	"github.com/pkg/errors"
	"golang.org/x/net/html"
)

const defaultGhtorrentMySQL = "http://ghtorrent-downloads.ewi.tudelft.nl/mysql/"

type dumpCommand struct {
	URL    string `short:"l" long:"url" description:"Link to GHTorrent MySQL dump in tar.gz format." default:"http://ghtorrent-downloads.ewi.tudelft.nl/mysql/." env:"GHTORRENT_MYSQL"`
	Stdin  bool   `long:"stdin" description:"read GHTorrent MySQL dump from stdin"`
	Output string `short:"o" long:"output" default:"data/repositories.csv.gz" description:"Ouput path for the gzipped file with the repositories extracted information."`
}

type repackCommand struct {
	dumpCommand
}

func (c *repackCommand) Execute(args []string) error {
	processDump(c.Stdin, c.URL, c.Output, repack)
	return nil
}

func processDump(stdin bool, url, output string, process func(*tar.Reader, io.Writer) int64) {
	startTime := time.Now()
	spin := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
	spin.Start()
	defer spin.Stop()

	input := dumpReader(stdin, url, spin)
	tracker := &trackingReader{RealReader: input, spin: spin}
	gzf, err := gzip.NewReader(tracker)
	if err != nil {
		fail("opening gzip stream", err)
	}
	defer gzf.Close()
	tarf := tar.NewReader(gzf)

	outf, err := createOutput(output)
	if err != nil {
		fail("creating output file", err)
	}

	gzw := gzip.NewWriter(outf)
	processed := process(tarf, gzw)
	fmt.Printf("\nRead      %s\nProcessed %s\nElapsed   %s\n",
		humanize.Bytes(uint64(tracker.TotalRead)), humanize.Bytes(uint64(processed)), time.Since(startTime))

	if err := gzw.Close(); err != nil {
		fail("closing output gz file", err)
	}

	if err := outf.Close(); err != nil {
		fail("closing output file", err)
	}
}

type trackingReader struct {
	RealReader io.ReadCloser
	TotalRead  int64
	spin       *spinner.Spinner
}

func (r *trackingReader) Read(p []byte) (n int, err error) {
	n, err = r.RealReader.Read(p)
	r.TotalRead += int64(n)
	if r.TotalRead%3 == 0 {
		// supposing that mod 3 is random, this is a quick and dirty way to (/ 3) updates.
		r.spin.Suffix = fmt.Sprintf(" %s", humanize.Bytes(uint64(r.TotalRead)))
	}

	return n, err
}

func (r *trackingReader) Close() error {
	return r.RealReader.Close()
}

func dumpReader(stdin bool, url string, spin *spinner.Spinner) io.ReadCloser {
	if stdin {
		fi, err := os.Stdin.Stat()
		if err != nil {
			fail("checking stat on stdin", err)
		}

		if fi.Mode()&os.ModeNamedPipe != 0 {
			return os.Stdin
		}
	}

	if url == defaultGhtorrentMySQL {
		spin.Suffix = " " + url
		url = findMostRecentMySQLDump(url)
	}

	fmt.Printf("\r>> %s\n", url)
	spin.Suffix = " connecting..."
	response, err := http.Get(url)
	if err != nil {
		fail("starting the download of "+url, err)
	}

	return response.Body
}

func findMostRecentMySQLDump(root string) string {
	ghturl, err := url.Parse(root)
	if err != nil {
		fail("parsing "+root, err)
	}

	response, err := http.Get(ghturl.String())
	if err != nil {
		fail("connecting to "+ghturl.String(), err)
	}
	defer response.Body.Close()

	tokenizer := html.NewTokenizer(response.Body)
	dumps := []string{}
	for token := tokenizer.Next(); token != html.ErrorToken; token = tokenizer.Next() {
		if token == html.StartTagToken {
			tag := tokenizer.Token()
			if tag.Data == "a" {
				for _, attr := range tag.Attr {
					if attr.Key == "href" {
						dumps = append(dumps, attr.Val)
						break
					}
				}
			}
		}
	}

	if len(dumps) == 0 {
		fail("getting the list of available dumps", errors.New("no dumps found"))
	}

	sort.Strings(dumps)
	lastDumpStr := dumps[len(dumps)-1]
	dumpurl, err := url.Parse(lastDumpStr)
	if err != nil {
		fail("parsing "+lastDumpStr, err)
	}

	return ghturl.ResolveReference(dumpurl).String()
}

func repack(r *tar.Reader, w io.Writer) int64 {
	const numTasks = 2
	var (
		processed int64
		status    int
		i         int
	)

	tarw := tar.NewWriter(w)
	for header, err := r.Next(); err != io.EOF; header, err = r.Next() {
		if err != nil {
			fail("reading tar.gz", err)
		}

		i++
		processed += header.Size
		isRelevant := strings.HasSuffix(header.Name, "watchers.csv") ||
			strings.HasSuffix(header.Name, "projects.csv")
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

			if _, err := io.Copy(tarw, r); err != nil {
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

	return processed
}

func createOutput(f string) (io.WriteCloser, error) {
	if err := os.MkdirAll(filepath.Dir(f), os.FileMode(0755)); err != nil {
		return nil, err
	}

	return os.Create(f)
}
