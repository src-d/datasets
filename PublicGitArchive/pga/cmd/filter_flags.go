package cmd

import (
	"fmt"

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

	ur, err := flags.GetString("url")
	if err != nil {
		return nil, err
	}

	f, err := filters.URLRegexp(ur)
	if err != nil {
		return nil, fmt.Errorf("invalid regular expression in -url: %v", err)
	}
	fs = append(fs, f)

	return filters.And(fs...), nil
}

func addFilterFlags(flags *pflag.FlagSet) {
	flags.StringSliceP("lang", "l", nil, "list of languages that the repositories should have")
	flags.StringP("url", "u", "", "regular expression that repo urls need to match")
}
