package cmd

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/src-d/datasets/PublicGitArchive/pga/pga"
	pb "gopkg.in/cheggaaa/pb.v1"
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get",
	Short: "gets all the repositories in the index",
	Long: `Downloads the repositories in the index, use flags to filter the results.

Alternatively, a list of .siva filenames can be passed through standard input.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := setupContext()

		source := urlFS("http://pga.sourced.tech/")

		dest, err := FileSystemFromFlags(cmd.Flags())
		if err != nil {
			return err
		}

		maxDownloads, err := cmd.Flags().GetInt("jobs")
		if err != nil {
			return err
		}

		var filenames []string

		stdin, err := cmd.Flags().GetBool("stdin")
		if err != nil {
			return err
		}
		if stdin {
			fmt.Fprintln(os.Stderr, "downloading siva files by name from stdin")
			fmt.Fprintln(os.Stderr, "filter flags will be ignored")
			b, err := ioutil.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("could not read from standard input: %v", err)
			}
			for _, filename := range strings.Split(string(b), "\n") {
				filename = strings.TrimSpace(filename)
				if filename == "" {
					continue
				}
				filenames = append(filenames, filename)
			}
		} else {
			f, err := getIndex(ctx)
			if err != nil {
				return fmt.Errorf("could not open index file: %v", err)
			}
			defer f.Close()

			index, err := pga.IndexFromCSV(f)
			if err != nil {
				return err
			}

			filter, err := filterFromFlags(cmd.Flags())
			if err != nil {
				return err
			}
			index = pga.WithFilter(index, filter)

			for {
				r, err := index.Next()
				if err == io.EOF {
					break
				} else if err != nil {
					return err
				}
				filenames = append(filenames, r.Filenames...)
			}
		}

		return downloadFilenames(ctx, dest, source, filenames, maxDownloads)
	},
}

func downloadFilenames(ctx context.Context, dest, source FileSystem,
	filenames []string, maxDownloads int) error {

	tokens := make(chan bool, maxDownloads)
	for i := 0; i < maxDownloads; i++ {
		tokens <- true
	}

	done := make(chan error)
	for _, filename := range filenames {
		filename := filepath.Join("siva", pgaVersion, filename[:2], filename)
		go func() {
			select {
			case <-tokens:
			case <-ctx.Done():
				done <- fmt.Errorf("canceled")
				return
			}
			defer func() { tokens <- true }()

			err := updateCache(ctx, dest, source, filename)
			if err != nil {
				fmt.Fprintf(os.Stderr, "could not get %s: %v\n", filename, err)
			}

			done <- err
		}()
	}

	bar := pb.StartNew(len(filenames))
	var finalErr error
	for i := 1; i <= len(filenames); i++ {
		if err := <-done; err != nil {
			finalErr = fmt.Errorf("there where failed downloads")
		}

		if finalErr == nil {
			bar.Set(i)
			bar.Update()
		}
	}

	return finalErr
}

func setupContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	var term = make(chan os.Signal)
	go func() {
		select {
		case <-term:
			logrus.Warningf("signal received, stopping...")
			cancel()
		}
	}()
	signal.Notify(term, syscall.SIGTERM, os.Interrupt)

	return ctx
}

func init() {
	RootCmd.AddCommand(getCmd)
	flags := getCmd.Flags()
	addFilterFlags(flags)
	flags.StringP("output", "o", ".", "path where the siva files should be stored")
	flags.IntP("jobs", "j", 10, "number of concurrent gets allowed")
	flags.BoolP("stdin", "i", false, "take list of siva files from standard input")
}
