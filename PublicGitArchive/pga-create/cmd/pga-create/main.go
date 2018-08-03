package main

import (
	"os"

	"github.com/jessevdk/go-flags"
)

var parser = flags.NewParser(nil, flags.Default)

func init() {
	if _, err := parser.AddCommand(
		"get-dataset",
		"Download Siva files from the given list in stdin.",
		"Download Siva files from the given list in stdin.",
		&getDatasetCommand{}); err != nil {
		panic(err)
	}

	if _, err := parser.AddCommand(
		"discover",
		"Fetch the GHTorrent MySQL dump and extract the list of repositories and the stars per repository.",
		"Fetch the GHTorrent MySQL dump and extract the list of repositories and the stars per repository.",
		&discoverCommand{}); err != nil {
		panic(err)
	}

	if _, err := parser.AddCommand(
		"select",
		"Reduce the full list of repositories from \"discover\" by the specified filters and write the result to stdout.",
		"Reduce the full list of repositories from \"discover\" by the specified filters and write the result to stdout.",
		&selectCommand{}); err != nil {
		panic(err)
	}

	if _, err := parser.AddCommand(
		"get-index",
		"Download the most recent dataset index file.",
		"Download the most recent dataset index file.",
		&getIndexCommand{}); err != nil {
		panic(err)
	}

	if _, err := parser.AddCommand(
		"index",
		"Create index.",
		"Create index.",
		&indexCommand{}); err != nil {
		panic(err)
	}

	if _, err := parser.AddCommand(
		"set-forks",
		"Set forks.",
		"Set forks.",
		&setForksCommand{}); err != nil {
		panic(err)
	}
}

func main() {
	if _, err := parser.Parse(); err != nil {
		if cerr, ok := err.(*flags.Error); ok && cerr.Type == flags.ErrHelp {
			os.Exit(0)
		}

		panic(err)
	}
}
