package cmd

import (
	"github.com/spf13/pflag"
	"github.com/src-d/datasets/PublicGitArchive/pga/pga"
	"github.com/src-d/datasets/PublicGitArchive/pga/pga/filters"
)

func filterFromFlags(flags *pflag.FlagSet) (pga.Filter, error) {
	langs, err := flags.GetStringSlice("lang")
	if err != nil {
		return nil, err
	}

	var fs []pga.Filter
	for _, lang := range langs {
		fs = append(fs, filters.HasLanguage(lang))
	}

	return filters.And(fs...), nil
}

func addFilterFlags(flags *pflag.FlagSet) {
	flags.StringSliceP("lang", "l", nil, "list of languages that the repositories should have")
}
