package model

import (
	"time"

	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-kallax.v1"
)

// Reference is a reference of a repository as present in our repository storage.
type Reference struct {
	ID                kallax.ULID `pk:""`
	kallax.Model      `table:"repository_references"`
	kallax.Timestamps `kallax:",inline"`
	// Name is the full reference name.
	Name string
	// Repository this reference belongs to.
	Repository *Repository `fk:",inverse"`
	// Hash is the hash of the reference.
	Hash SHA1
	// Init is the hash of the init commit reached from this reference.
	Init SHA1
	// Roots is a slice of the hashes of all root commits reachable from
	// this reference.
	Roots SHA1List
	// Time is the time of the commit this reference points too.
	Time time.Time `kallax:"reference_time"`
}

func newReference() *Reference {
	return &Reference{ID: kallax.NewULID()}
}

// GitReference returns a git reference for this instance.
func (r *Reference) GitReference() *plumbing.Reference {
	return plumbing.NewHashReference(
		plumbing.ReferenceName(r.Name),
		plumbing.Hash(r.Hash),
	)
}
