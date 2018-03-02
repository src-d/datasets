package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/src-d/datasets/PublicGitArchive/pga/pga"
	"golang.org/x/sync/errgroup"
	pb "gopkg.in/cheggaaa/pb.v1"
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get",
	Short: "gets all the repositories in the index",
	Long: `Downloads the repositories in the index, use flags to filter the results.

Alternatively, a list of .siva filenames can be passed through standard input.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dest, err := FileSystemFromFlags(cmd.Flags())
		if err != nil {
			return err
		}

		maxDownloads, err := cmd.Flags().GetInt("jobs")
		if err != nil {
			return err
		}

		var index pga.Index

		stdin, err := cmd.Flags().GetBool("stdin")
		if err != nil {
			return err
		}
		if stdin {
			fmt.Fprintln(os.Stderr, "downloading siva files by name from stdin")
			fmt.Fprintln(os.Stderr, "filter flags will be ignored")
			index = getIndexFromStdin(os.Stdin)
		} else {
			f, err := getIndex()
			if err != nil {
				return fmt.Errorf("could not open index file: %v", err)
			}
			defer f.Close()

			index, err = pga.IndexFromCSV(f)
			if err != nil {
				return err
			}

			filter, err := filterFromFlags(cmd.Flags())
			if err != nil {
				return err
			}
			index = pga.WithFilter(index, filter)
		}

		tokens := make(chan bool, maxDownloads)
		for i := 0; i < maxDownloads; i++ {
			tokens <- true
		}

		var jobsDone int
		var mutex sync.Mutex
		withMutex := func(f func()) { mutex.Lock(); f(); mutex.Unlock() }
		var totalJobs int

		var group errgroup.Group
		for {
			r, err := index.Next()
			if err == io.EOF {
				break
			} else if err != nil {
				return err
			}

			totalJobs += len(r.Filenames)

			for _, filename := range r.Filenames {
				filename := filename // avoid data race
				group.Go(func() error {
					<-tokens
					defer func() {
						withMutex(func() { jobsDone++ })
						tokens <- true
					}()
					if err := get(dest, filename); err != nil {
						fmt.Fprintf(os.Stderr, "could not get %s: %v\n", filename, err)
					}
					return nil
				})
			}
		}

		done := make(chan struct{})
		go func() { group.Wait(); close(done) }()

		bar := pb.StartNew(totalJobs)
		tick := time.Tick(time.Second)
		for {
			select {
			case <-done:
				withMutex(func() { bar.Set(totalJobs) })
				bar.Update()
				return nil
			case <-tick:
				withMutex(func() { bar.Set(jobsDone) })
			}
		}
	},
}

type stdinIndex struct{ *bufio.Scanner }

func (i stdinIndex) Next() (*pga.Repository, error) {
	if !i.Scan() {
		if i.Err() == nil {
			return nil, io.EOF
		}
		return nil, i.Err()
	}
	s := i.Text()
	if !strings.HasSuffix(s, ".siva") {
		return nil, fmt.Errorf("expected siva filename, got %s", s)
	}
	fs := strings.Split(s, ",")
	return &pga.Repository{Filenames: fs}, nil
}

func getIndexFromStdin(r io.Reader) pga.Index {
	return stdinIndex{bufio.NewScanner(r)}
}

func get(dest FileSystem, name string) error {
	source := urlFS("http://pga.sourced.tech/")
	ok, err := copy(dest, source, filepath.Join("siva", "latest", name[:2], name))
	if !ok {
		return err
	} else if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	return nil
}

func init() {
	RootCmd.AddCommand(getCmd)
	flags := getCmd.Flags()
	addFilterFlags(flags)
	flags.StringP("output", "o", ".", "path where the siva files should be stored")
	flags.IntP("jobs", "j", 10, "number of concurrent gets allowed")
	flags.BoolP("stdin", "i", false, "take list of siva files from standard input")
}
