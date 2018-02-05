package model

import "gopkg.in/src-d/go-kallax.v1"

// RepositoryGroup represents a set of repositories identified by a label
// with a main repository.
// For example, a repository and all its forks is a repository group with all
// forks as repositories and the original one as MainRepository.
type RepositoryGroup struct {
	Label          string
	MainRepository kallax.ULID
	Repositories   []kallax.ULID
}
