package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/src-d/datasets/PublicGitArchive/pga/pga"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "list all the repositories in the index",
	Long:  `List the repositories in the index, use flags to filter the results.`,
	RunE: func(cmd *cobra.Command, args []string) error {
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
			fmt.Println(r.URL)
		}
		return nil
	},
}

func init() {
	RootCmd.AddCommand(listCmd)
	addFilterFlags(listCmd.Flags())
}
