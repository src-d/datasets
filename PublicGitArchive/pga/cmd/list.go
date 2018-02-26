package cmd

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"

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

		format, err := cmd.Flags().GetString("format")
		if err != nil {
			return err
		}
		var formatter func(r *pga.Repository) (string, error)
		switch format {
		case "url":
			formatter = func(r *pga.Repository) (string, error) { return r.URL + "\n", nil }
		case "json":
			formatter = func(r *pga.Repository) (string, error) {
				b, err := json.Marshal(r)
				return string(b) + "\n", err
			}
		case "csv":
			formatter = func(r *pga.Repository) (string, error) {
				buf := new(bytes.Buffer)
				w := csv.NewWriter(buf)
				err := w.Write(r.ToCSV())
				w.Flush()
				return buf.String(), err
			}
		default:
			return fmt.Errorf("unkown format in --format %q", format)
		}

		index = pga.WithFilter(index, filter)
		for {
			r, err := index.Next()
			if err == io.EOF {
				break
			} else if err != nil {
				return err
			}

			if s, err := formatter(r); err != nil {
				fmt.Fprintf(os.Stderr, "could not format repository %s: %v\n", r.URL, err)
			} else {
				fmt.Print(s)
			}
		}
		return nil
	},
}

func init() {
	RootCmd.AddCommand(listCmd)
	flags := listCmd.Flags()
	addFilterFlags(flags)
	flags.StringP("format", "f", "url", "format of the output (url, csv, or json)")
}
