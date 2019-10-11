// Package pga provides a simple API to access the Public Git Archive repository.
// For more information check http://pga.sourced.tech/.
package pga

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
)

// Repository provides abstraction for the data in the indexes.
type Repository interface {
	ToCSV() []string
	GetURL() string
	GetLanguages() []string
	GetFilenames() []string
}

// A Filter provides a way to filter repositories.
type Filter func(Repository) bool

// Dataset provides abstraction for creating Repositories from a CSV file
type Dataset interface {
	Name() string
	ReadHeader(columnNames []string) error
	RepositoryFromTuple(cols []string) (repo Repository, err error)
}

// ForEachRepository applies a function to each of the rows of a CSV index.
func ForEachRepository(ctx context.Context, r *csv.Reader, dataset Dataset, filter Filter, f func(r Repository) error) error {
	if columnNames, err := r.Read(); err != nil {
		return fmt.Errorf("could not read headers row: %v", err)
	} else if err = dataset.ReadHeader(columnNames); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return &CommandCanceledError{}
		default:
		}
		cols, err := r.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		repository, err := dataset.RepositoryFromTuple(cols)
		if err != nil {
			return err
		}
		if filter(repository) {
			err = f(repository)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Datasets is a slice containing Dataset objects on which we can apply the `get` and `list` commands.
var Datasets = []Dataset{
	&SivaDataset{},
	&UastDataset{},
}

type badHeaderLengthError struct {
	expectedMin int
	expectedMax int
	length      int
}

func (e *badHeaderLengthError) Error() string {
	if e.expectedMin == e.expectedMax {
		return fmt.Sprintf("bad header length: expected  %d but got %d", e.expectedMin, e.length)
	}
	return fmt.Sprintf("bad header length: expected  %d to %d but got %d", e.expectedMin, e.expectedMax, e.length)
}

type badHeaderColumnError struct {
	expected string
	index    int
	col      string
}

func (e *badHeaderColumnError) Error() string {
	return fmt.Sprintf("bad header column: expected  %s at index %d but got %s", e.expected, e.index, e.col)
}

// CommandCanceledError is raised if the running command is canceled
type CommandCanceledError struct{}

func (e *CommandCanceledError) Error() string {
	return "command canceled"
}
