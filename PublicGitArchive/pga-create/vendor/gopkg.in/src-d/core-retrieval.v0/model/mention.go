package model

import "gopkg.in/src-d/go-kallax.v1"

//go:generate kallax gen

func newMention() *Mention {
	return &Mention{ID: kallax.NewULID()}
}

// Mention is the sighting of a remote repository online.
type Mention struct {
	ID                kallax.ULID `pk:""`
	kallax.Model      `table:"mentions"`
	kallax.Timestamps `kallax:",inline"`
	// Endpoint is the repository URL as found.
	Endpoint string
	// Aliases are all the endpoints obtained from this mention. Endpoint field should be also included
	Aliases []string
	// IsFork is set to true if the repository is known to be fork. It is set to nil if the provider does not provide
	// this information at all. Note that false means that the repository is not a known fork to the provider, but it
	// might still be a fork, for example, a fork in GitHub from an original repository in Bitbucket.
	IsFork *bool
	// Provider is the repository provider (e.g. github).
	Provider string
	// VCS contains the version control system of this Mention.
	VCS VCS
}

type VCS string

const (
	// Git version control system
	GIT VCS = "git"
)
