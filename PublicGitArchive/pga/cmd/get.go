package cmd

import (
	"context"
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	pb "github.com/cheggaaa/pb/v3"
	"github.com/spf13/cobra"
	"github.com/src-d/datasets/PublicGitArchive/pga/pga"
)

const rootURL = "http://pga.sourced.tech"

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get",
	Short: "gets all the repositories in the index",
	Long: `Downloads the repositories in the index, use flags to filter the results.

Alternatively, a list of .siva filenames can be passed through standard input.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dataset, err := handleDatasetArg(cmd.Use, cmd.Flags())
		if err != nil {
			return err
		}
		ctx := setupContext()
		source := urlFS(rootURL)
		dest, err := FileSystemFromFlags(cmd.Flags())
		if err != nil {
			return err
		}
		maxDownloads, err := cmd.Flags().GetInt("jobs")
		if err != nil {
			return err
		}
		var filenames = map[string]struct{}{}
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
				filenames[filename] = struct{}{}
			}
		} else {
			f, err := getIndex(ctx, dataset.Name())
			if err != nil {
				return fmt.Errorf("could not open index file: %v", err)
			}
			defer f.Close()
			r := csv.NewReader(f)
			err = dataset.ReadHeader(r)
			if err != nil {
				return err
			}
			filter, err := filterFromFlags(cmd.Flags())
			if err != nil {
				return err
			}
			addFiles := func(r pga.Repository) error {
				for _, filename := range r.GetFilenames() {
					filenames[filename] = struct{}{}
				}
				return nil
			}
			dataset.ForEach(ctx, r, filter, addFiles)
		}

		return downloadFilenames(ctx, dest, source, dataset.Name(), filenames, maxDownloads)
	},
}

func downloadFilenames(ctx context.Context, dest, source FileSystem, datasetName string,
	filenames map[string]struct{}, maxDownloads int) error {

	tokens := make(chan bool, maxDownloads)
	for i := 0; i < maxDownloads; i++ {
		tokens <- true
	}

	done := make(chan error)
	for filename := range filenames {
		filename := filepath.Join(datasetName, pgaVersion, filename[:2], filename)
		go func() {
			select {
			case <-tokens:
			case <-ctx.Done():
				return
			}
			defer func() { tokens <- true }()

			err := updateCache(ctx, dest, source, filename)
			if _, cancel := err.(*pga.CommandCanceledError); err != nil && !cancel {
				err = fmt.Errorf("could not get %s: %v", filename, err)
			}
			done <- err
		}()
	}

	bar := pb.StartNew(len(filenames))
	var err error
	for i := 1; i <= len(filenames); i++ {
		err = <-done
		if err != nil {
			if _, cancel := err.(*pga.CommandCanceledError); !cancel {
				err = fmt.Errorf("there where failed downloads: %s", err)
			}
			break
		}
		bar.Increment()
	}
	bar.Finish()
	return err
}

func init() {
	RootCmd.AddCommand(getCmd)
	flags := getCmd.Flags()
	addFilterFlags(flags)
	flags.StringP("output", "o", ".", "path where the siva files should be stored")
	flags.IntP("jobs", "j", 10, "number of concurrent gets allowed")
	flags.BoolP("stdin", "i", false, "take list of siva files from standard input")
}
