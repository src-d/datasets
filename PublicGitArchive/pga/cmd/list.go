package cmd

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
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
		ctx := setupContext()
		f, err := getIndex(ctx)
		if err != nil {
			return fmt.Errorf("could not open index file: %v", err)
		}

		index, err := pga.IndexFromCSV(f)
		if err != nil {
			_ = f.Close()
			return err
		}

		filter, err := filterFromFlags(cmd.Flags())
		if err != nil {
			_ = f.Close()
			return err
		}

		formatter, err := formatterFromFlags(cmd.Flags())
		if err != nil {
			_ = f.Close()
			return err
		}

		index = pga.WithFilter(index, filter)
		for {
			select {
			case <-ctx.Done():
				_ = f.Close()
				return fmt.Errorf("command canceled")
			default:
			}

			r, err := index.Next()
			if err == io.EOF {
				break
			} else if err != nil {
				_ = f.Close()
				return err
			}

			if s, err := formatter(r); err != nil {
				fmt.Fprintf(os.Stderr, "could not format repository %s: %v\n", r.URL, err)
			} else {
				fmt.Print(s)
			}
		}

		return f.Close()
	},
}

func formatterFromFlags(flags *pflag.FlagSet) (func(*pga.Repository) (string, error), error) {
	format, err := flags.GetString("format")
	if err != nil {
		return nil, err
	}

	switch format {
	case "url":
		return func(r *pga.Repository) (string, error) {
			return r.URL + "\n", nil
		}, nil
	case "json":
		return func(r *pga.Repository) (string, error) {
			b, err := json.Marshal(r)
			return string(b) + "\n", err
		}, nil
	case "csv":
		return func(r *pga.Repository) (string, error) {
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
