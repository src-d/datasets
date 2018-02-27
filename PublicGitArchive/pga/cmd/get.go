package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
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
	Long:  `Downloads the repositories in the index, use flags to filter the results.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		filter, err := filterFromFlags(cmd.Flags())
		if err != nil {
			return err
		}

		dest, err := destinationFromFlags(cmd.Flags())
		if err != nil {
			return err
		}

		maxDownloads, err := cmd.Flags().GetInt("jobs")
		if err != nil {
			return err
		}

		f, err := getIndex()
		if err != nil {
			return fmt.Errorf("could not open index file: %v", err)
		}
		defer f.Close()
		index, err := pga.IndexFromCSV(f)
		if err != nil {
			return err
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
		index = pga.WithFilter(index, filter)
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
				withMutex(func() { bar.Set(jobsDone) })
				return nil
			case <-tick:
				withMutex(func() { bar.Set(jobsDone) })
			}
		}
	},
}

func get(dest *destination, name string) error {
	wc, err := dest.newWriter(name)
	if err != nil {
		return fmt.Errorf("could not create a new file in destination %s: %v", dest, err)
	}

	url := fmt.Sprintf("http://pga.sourced.tech/siva/latest/%s/%s", name[:2], name)
	res, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("could not get %s: %v", url, err)
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("could not get %s: %s", url, res.Status)
	}

	_, err = io.Copy(wc, res.Body)
	if err != nil {
		wc.Close()
		return fmt.Errorf("could not copy to %s: %v", dest, err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("could not close %s: %v", dest, err)
	}
	return nil
}

func init() {
	RootCmd.AddCommand(getCmd)
	addFilterFlags(getCmd.Flags())
	getCmd.Flags().StringP("output", "o", ".", "path where the siva files should be stored")
	getCmd.Flags().IntP("jobs", "j", 10, "number of concurrent gets allowed")
}