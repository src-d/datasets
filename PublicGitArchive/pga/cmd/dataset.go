package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/pflag"
	"github.com/src-d/datasets/PublicGitArchive/pga/pga"
)

func handleDatasetArg(cmd string, flags *pflag.FlagSet) (pga.Dataset, error) {
	if flags.NArg() != 1 {
		return nil, fmt.Errorf("usage: pga list <dataset>")
	}
	datasetName := flags.Arg(0)
	for _, dataset := range pga.Datasets {
		if datasetName == dataset.Name() {
			return dataset, nil
		}
	}
	knownDatasets := make([]string, 0, len(pga.Datasets))
	for _, dataset := range pga.Datasets {
		knownDatasets = append(knownDatasets, dataset.Name())
	}
	sort.Strings(knownDatasets)
	return nil, fmt.Errorf("unknown dataset: %s (choose from %s)", datasetName, strings.Join(knownDatasets, ", "))
}
