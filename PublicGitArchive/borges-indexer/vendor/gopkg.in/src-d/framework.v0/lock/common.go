// Package lock provides implementations for cancellable and distributed locks.
package lock

import (
	"net/url"
	"time"

	"gopkg.in/src-d/go-errors.v0"
)

var (
	ErrUnsupportedService      = errors.NewKind("unsupported service: %s")
	ErrInvalidConnectionString = errors.NewKind("invalid connection string: %s")
	ErrCanceled                = errors.NewKind("context canceled")
	ErrAlreadyClosed           = errors.NewKind("already closed")
)

// Services is a registry of all supported services by name. Map key is the
// service name, which will be looked up in URIs scheme.
var Services map[string]func(string) (Service, error)

func init() {
	Services = make(map[string]func(string) (Service, error))
}

// New creates a service given a connection string.
func New(connstr string) (Service, error) {
	u, err := url.Parse(connstr)
	if err != nil {
		return nil, ErrInvalidConnectionString.Wrap(err, "invalid URL")
	}

	name := u.Scheme
	srvf, ok := Services[name]
	if !ok {
		return nil, ErrUnsupportedService.New(name)
	}

	return srvf(connstr)
}

// SessionConfig holds configuration for a locking session.
type SessionConfig struct {
	// Timeout is the timeout when acquiring a lock. Calls to Lock() on a Locker
	// in the session will fail if the lock cannot be acquired before timeout.
	Timeout time.Duration
	// TTL is the time-to-live of all locks in a session. A lock operation times
	// out when the TTL expires. A lock is lost whenever it cannot be kept alive
	// inside the TTL. For example, a lock in a distributed lock service maintains
	// a keep alive heartbeat, once a heartbeat is not received for more than
	// the specified TTL, it must be assumed that the lock is no longer held.
	TTL time.Duration
}

// Service is a locking service.
type Service interface {
	// NewSession creates a new locking session with the given configuration.
	// An error is returned if the session cannot be created (e.g. invalid
	// configuration, connection cannot be established to remote service).
	NewSession(*SessionConfig) (Session, error)
	// Close closes the service and releases any resources held by it.
	// If it is called more than once, ErrAlreadyClosed is returned.
	Close() error
}

// Session is a locking session that can be reused to get multiple locks.
// Multiple actors should use different sessions.
type Session interface {
	// NewService creates a NewService for the given id. Lockers returned for the
	// same id on different sessions are mutually exclusive.
	NewLocker(id string) Locker
	// Close closes the session and releases any resources held by it.
	// If it is called more than once, ErrAlreadyClosed is returned.
	Close() error
}

// Locker is the interface to a lock.
type Locker interface {
	// Context can be used to cancel the operation (e.g. with a timeout).
	// If it succeeds, it returns a cancel channel. Otherwise, it returns an
	// error. The cancel channel is closed whenever the lock is not held anymore.
	// Note that the lock might be lost before calling to Unlock.
	Lock() (<-chan struct{}, error)
	// Unlock releases the lock. It might return an error if the lock could not
	// be released cleanly. However, this error is merely informative and no
	// action needs to be taken. Locking services must ensure that a lock that
	// failed to be released cleanly expires at some point (e.g. after session
	// TTL expires).
	Unlock() error
}
