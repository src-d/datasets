package model

import (
	"time"

	"gopkg.in/src-d/go-kallax.v1"
)

//go:generate kallax gen

func newRepository() *Repository {
	return &Repository{ID: kallax.NewULID(), Status: Pending}
}

// Repository represents a remote repository found on the Internet.
type Repository struct {
	ID                kallax.ULID `pk:""`
	kallax.Model      `table:"repositories"`
	kallax.Timestamps `kallax:",inline"`
	// Endpoints is a slice of valid git endpoints to reach this repository.
	// For example, git://host/my/repo.git and https://host/my/repo.git.
	// They are meant to be endpoints of the same exact repository, and not
	// mirrors.
	Endpoints []string
	// Status is the fetch status of tge repository in our repository storage.
	Status FetchStatus
	// FetchedAt is the timestamp of the last time this repository was
	// fetched and archived in our repository storage successfully.
	FetchedAt *time.Time
	// FetchErrorAt is the timestamp of the last fetch error, if any.
	FetchErrorAt *time.Time
	// LastCommitAt is the last commit time found in this repository.
	LastCommitAt *time.Time
	// References is the current slice of references as present in our
	// repository storage.
	References []*Reference
	// IsFork stores if this repository is a fork or not. It can be nil if we don't know.
	IsFork *bool
}

// FetchStatus represents the fetch status of this repository.
type FetchStatus string

const (
	// NotFound means that the remote repository was not found at the given
	// endpoints.
	NotFound FetchStatus = "not_found"
	// Fetched means that the remote repository was found, fetched and
	// successfully stored.
	Fetched FetchStatus = "fetched"
	// Pending is the default value, meaning that the repository has not
	// been fetched yet.
	Pending FetchStatus = "pending"
	// Fetching means the remote repository was found and started being
	// fetched. It could also mean that there was an error and the repository
	// never finished fetching.
	Fetching FetchStatus = "fetching"
	// AuthRequired means the remote repository returns an authentication required
	// error when you try to fetch it. It doesn't mean that repository exists,
	// but if so, it cannot be processed without appropiate credentials.
	AuthRequired FetchStatus = "auth_req"
)

// Language represents a language name.
type Language string
