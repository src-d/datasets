package pga

import (
	"fmt"
	"strconv"
	"strings"
)

type parser struct {
	cols       []string
	err        error
	csvHeaders *[]string
}

func (p *parser) readString(idx int) string { return p.cols[idx] }

func (p *parser) readStringList(idx int) []string {
	s := p.cols[idx]
	if s == "" {
		return nil
	}
	return strings.Split(p.cols[idx], ",")
}

func (p *parser) readInt(idx int) int64 {
	if p.err != nil {
		return 0
	}
	s := p.cols[idx]
	if s == "" {
		return 0
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		p.err = fmt.Errorf("parsing %s integer %q: %v", (*p.csvHeaders)[idx], s, err)
	}
	return v
}

func (p *parser) readIntList(idx int) []int64 {
	if p.err != nil {
		return nil
	}
	ts := p.readStringList(idx)
	vs := make([]int64, len(ts))
	for i, t := range ts {
		v, err := strconv.ParseInt(t, 10, 64)
		if err != nil {
			p.err = fmt.Errorf("could not parse %q in %s: %v", t, (*p.csvHeaders)[idx], err)
			return nil
		}
		vs[i] = v
	}
	return vs
}

func (p *parser) readFloat(idx int) float64 {
	if p.err != nil {
		return 0
	}
	s := p.cols[idx]
	if s == "" {
		return 0
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		p.err = fmt.Errorf("parsing %s integer %q: %v", (*p.csvHeaders)[idx], s, err)
	}
	return v
}

func (p *parser) readFloatList(idx int) []float64 {
	if p.err != nil {
		return nil
	}
	ts := p.readStringList(idx)
	vs := make([]float64, len(ts))
	for i, t := range ts {
		v, err := strconv.ParseFloat(t, 64)
		if err != nil {
			p.err = fmt.Errorf("could not parse %q in %s: %v", t, (*p.csvHeaders)[idx], err)
			return nil
		}
		vs[i] = v
	}
	return vs
}
