// Package cmd contains all of the subcommands available in the pga tool.
package cmd

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "pga",
	Short: "The Public Git Archive exploration and download tool",
	Long: `pga allows you to list, filterm and download files from the Public Git Archive dataset.

For more info, check http://pga.sourced.tech/`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		v, err := cmd.Flags().GetBool("verbose")
		if err != nil {
			return err
		}
		if v {
			logrus.SetLevel(logrus.DebugLevel)
		}

		pv, err := cmd.Flags().GetString("pga-version")
		if err != nil {
			return err
		}

		pgaVersion = pv
		indexName = pv + ".csv.gz"

		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var (
	indexName  string
	pgaVersion string
)

func init() {
	RootCmd.Flags().BoolP("toggle", "t", false, "help message for toggle")
	RootCmd.PersistentFlags().BoolP("verbose", "v", false, "log more information")
	RootCmd.PersistentFlags().StringVar(&pgaVersion, "pga-version", "latest", "pga version to be used")
}
