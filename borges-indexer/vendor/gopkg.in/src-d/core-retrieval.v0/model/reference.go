package model

import (
	"time"

	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-kallax.v1"
)

// Reference is a reference of a repository as present in our repository storage.
type Reference struct {
	kallax.Timestamps `kallax:",inline"`
	// Name is the full reference name.
	Name string
	// Hash is the hash of the reference.
	Hash SHA1
	// Init is the hash of the init commit reached from this reference.
	Init SHA1
	// Roots is a slice of the hashes of all root commits reachable from
	// this reference.
	Roots []SHA1
	// Time is the time of the commit this reference points too.
	Time time.Time
}

func (r *Reference) GitReference() *plumbing.Reference {
	return plumbing.NewHashReference(
		plumbing.ReferenceName(r.Name),
		plumbing.Hash(r.Hash),
	)
}
