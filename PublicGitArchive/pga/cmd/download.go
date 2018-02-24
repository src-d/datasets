package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/src-d/datasets/PublicGitArchive/pga/pga"
)

// downloadCmd represents the download command
var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "downloads all the repositories in the index",
	Long:  `Downloads the repositories in the index, use flags to filter the results.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		f, err := os.Open("/Users/francesc/Downloads/latest.csv")
		if err != nil {
			return fmt.Errorf("could not open index file: %v", err)
		}
		defer f.Close()

		dest, err := cmd.Flags().GetString("output")
		if err != nil {
			return err
		}

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

			for _, filename := range r.Filenames {
				if err := download(dest, filename); err != nil {
					return fmt.Errorf("could not download %s: %v", filename, err)
				}
			}

			fmt.Println(r.URL)
		}
		return nil
	},
}

func download(dest, name string) error {
	dir := filepath.Join(dest, "siva", "latest", name[:2])
	if err := os.MkdirAll(dir, 0777); err != nil {
		return fmt.Errorf("could not create destination directory: %v", err)
	}

	path := filepath.Join(dest, "siva", "latest", name[:2], name)
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("could not create %s: %v", path, err)
	}

	url := fmt.Sprintf("http://pga.sourced.tech/siva/latest/%s/%s", name[:2], name)
	res, err := http.Get(url)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf(res.Status)
	}

	_, err = io.Copy(f, res.Body)
	if err != nil {
		f.Close()
		return fmt.Errorf("could not copy to %s: %v", path, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("could not close %s: %v", path, err)
	}
	return nil
}

func init() {
	RootCmd.AddCommand(downloadCmd)
	addFilterFlags(downloadCmd.Flags())
	downloadCmd.Flags().StringP("output", "o", ".", "path where the siva files should be stored")
}
