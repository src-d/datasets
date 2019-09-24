package cmd

import (
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"
)

var parser = flags.NewParser(nil, flags.Default)

func init() {
	if _, err := parser.AddCommand(
		"repack",
		"Repack GHTorrent MySQL dump",
		"Repack GHTorrent MySQL dump",
		&repackCommand{}); err != nil {
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

func Main() {
	if _, err := parser.Parse(); err != nil {
		if cerr, ok := err.(*flags.Error); ok && cerr.Type == flags.ErrHelp {
			os.Exit(0)
		}

		panic(err)
	}
}

func fail(operation string, err error) {
	fmt.Fprintf(os.Stderr, "Error: %s: %s\n", operation, err.Error())
	os.Exit(1)
}
