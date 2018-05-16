package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

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
			f, err := getIndex()
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

		return downloadFilenames(dest, source, filenames, maxDownloads)
	},
}

func downloadFilenames(dest, source FileSystem, filenames []string, maxDownloads int) error {
	tokens := make(chan bool, maxDownloads)
	for i := 0; i < maxDownloads; i++ {
		tokens <- true
	}

	done := make(chan bool)
	for _, filename := range filenames {
		filename := filepath.Join("siva", "latest", filename[:2], filename)
		go func() {
			<-tokens
			defer func() { tokens <- true }()

			if err := updateCache(dest, source, filename); err != nil {
				fmt.Fprintf(os.Stderr, "could not get %s: %v\n", filename, err)
			}
			done <- true
		}()
	}

	bar := pb.StartNew(len(filenames))
	for i := 1; ; i++ {
		<-done
		bar.Set(i)
		bar.Update()
		if i == len(filenames) {
			return nil
		}
	}
}

func init() {
	RootCmd.AddCommand(getCmd)
	flags := getCmd.Flags()
	addFilterFlags(flags)
	flags.StringP("output", "o", ".", "path where the siva files should be stored")
	flags.IntP("jobs", "j", 10, "number of concurrent gets allowed")
	flags.BoolP("stdin", "i", false, "take list of siva files from standard input")
}
