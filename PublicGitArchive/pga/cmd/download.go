package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/src-d/datasets/PublicGitArchive/pga/pga"
	"golang.org/x/sync/errgroup"
	pb "gopkg.in/cheggaaa/pb.v1"
)

// downloadCmd represents the download command
var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "downloads all the repositories in the index",
	Long:  `Downloads the repositories in the index, use flags to filter the results.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		f, err := getIndex()
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

		maxDownloads, err := cmd.Flags().GetInt("jobs")
		if err != nil {
			return err
		}
		tokens := make(chan bool, maxDownloads)
		for i := 0; i < maxDownloads; i++ {
			tokens <- true
		}

		var jobs struct {
			total int
			done  int
			sync.Mutex
		}

		var group errgroup.Group
		index = pga.WithFilter(index, filter)
		for {
			r, err := index.Next()
			if err == io.EOF {
				break
			} else if err != nil {
				return err
			}

			jobs.Lock()
			jobs.total += len(r.Filenames)
			jobs.Unlock()

			for _, filename := range r.Filenames {

				filename := filename // avoid data race
				group.Go(func() error {
					<-tokens
					defer func() {
						jobs.Lock()
						jobs.done++
						jobs.Unlock()
						tokens <- true
					}()
					if err := download(dest, filename); err != nil {
						return fmt.Errorf("could not download %s: %v", filename, err)
					}

					return nil
				})
			}
		}
		errc := make(chan error, 1)
		go func() { errc <- group.Wait() }()

		jobs.Lock()
		bar := pb.StartNew(jobs.total)
		jobs.Unlock()
		tick := time.Tick(time.Second)
		for {
			select {
			case err := <-errc:
				return err
			case <-tick:
				jobs.Lock()
				bar.Set(jobs.done)
				jobs.Unlock()
			}
		}
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
	downloadCmd.Flags().IntP("jobs", "j", 10, "number of concurrent downloads allowed")
}
