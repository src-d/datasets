package cmd

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/src-d/datasets/PublicGitArchive/pga/pga"
)

// sivaCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "list all the repositories in the index",
	Long:  `List the repositories in the index, use flags to filter the results.`,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		dataset, err := handleDatasetArg(cmd.Use, cmd.Flags())
		if err != nil {
			return err
		}
		ctx := setupContext()
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
		formatter, err := formatterFromFlags(cmd.Flags())
		if err != nil {
			return err
		}
		printRepository := func(r pga.Repository) error {
			if s, err := formatter(r); err != nil {
				fmt.Fprintf(os.Stderr, "could not format repository %s: %v\n", r.GetURL(), err)
			} else {
				fmt.Print(s)
			}
			return nil
		}
		return dataset.ForEach(ctx, r, filter, printRepository)
	},
}

func formatterFromFlags(flags *pflag.FlagSet) (func(pga.Repository) (string, error), error) {
	format, err := flags.GetString("format")
	if err != nil {
		return nil, err
	}
	switch format {
	case "url":
		return func(r pga.Repository) (string, error) {
			return r.GetURL() + "\n", nil
		}, nil
	case "json":
		return func(r pga.Repository) (string, error) {
			b, err := json.Marshal(r)
			return string(b) + "\n", err
		}, nil
	case "csv":
		return func(r pga.Repository) (string, error) {
			buf := new(bytes.Buffer)
			w := csv.NewWriter(buf)
			err := w.Write(r.ToCSV())
			w.Flush()
			return buf.String(), err
		}, nil
	default:
		return nil, fmt.Errorf("unkown format in --format %q", format)
	}
}

func init() {
	RootCmd.AddCommand(listCmd)
	flags := listCmd.Flags()
	addFilterFlags(flags)
	flags.StringP("format", "f", "url", "format of the output (url, csv, or json)")
}
