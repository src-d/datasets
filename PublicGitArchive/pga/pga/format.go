package pga

import (
	"strconv"
	"strings"
)

func formatStringList(l []string) string { return strings.Join(l, ",") }

func formatInt(v int64) string { return strconv.FormatInt(v, 10) }

func formatIntList(vs []int64) string {
	ts := make([]string, len(vs))
	for i, v := range vs {
		ts[i] = formatInt(v)
	}
	return formatStringList(ts)
}

func formatFloat(v float64) string { return strconv.FormatFloat(v, 'f', 2, 64) }

func formatFloatList(vs []float64) string {
	ts := make([]string, len(vs))
	for i, v := range vs {
		ts[i] = formatFloat(v)
	}
	return formatStringList(ts)
}
