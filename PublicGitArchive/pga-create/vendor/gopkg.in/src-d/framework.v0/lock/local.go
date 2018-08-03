package lock

import (
	"sync"
	"time"
)

const ServiceLocal = "local"

func init() {
	Services[ServiceLocal] = func(string) (Service, error) {
		return NewLocal(), nil
	}
}

type localLock struct {
	ch chan struct{}
}

func newLocalLock() *localLock {
	l := &localLock{make(chan struct{}, 1)}
	l.ch <- struct{}{}
	return l
}

func (l *localLock) Lock(timeout time.Duration) bool {
	if timeout == 0 {
		<-l.ch
		return true
	}

	select {
	case _, ok := <-l.ch:
		return ok
	case <-time.After(timeout):
		return false
	}
}

func (l *localLock) Unlock() {
	l.ch <- struct{}{}
}

func (l *localLock) Close() error {
	if l.ch == nil {
		return ErrAlreadyClosed.New()
	}

	close(l.ch)
	l.ch = nil
	return nil
}

type localSrv struct {
	locks    map[string]*localLock
	refCount map[string]int
	m        *sync.Mutex
	closed   bool
}

// NewLocal creates a new locking service that uses in-process locks. This can
// be used whenever locking is relevant only to the local process. Local locks
// are never lost, so TTL is ignored.
func NewLocal() Service {
	return &localSrv{
		locks:    map[string]*localLock{},
		refCount: map[string]int{},
		m:        &sync.Mutex{},
	}
}

func (s *localSrv) NewSession(cfg *SessionConfig) (Session, error) {
	return &localSess{
		cfg: cfg,
		srv: s,
	}, nil
}

func (s *localSrv) Close() error {
	s.m.Lock()
	defer s.m.Unlock()

	if s.closed {
		return ErrAlreadyClosed.New()
	}

	var err error
	for _, lock := range s.locks {
		cerr := lock.Close()
		if cerr != nil && err == nil {
			err = cerr
		}
	}

	s.closed = true
	return err
}

func (s *localSrv) getLock(id string) *localLock {
	s.m.Lock()
	defer s.m.Unlock()

	lock, ok := s.locks[id]
	if !ok {
		lock = newLocalLock()
		s.locks[id] = lock
	}

	s.refCount[id] = s.refCount[id] + 1
	return lock
}

func (s *localSrv) freeLock(id string) {
	s.m.Lock()
	defer s.m.Unlock()

	c, ok := s.refCount[id]
	c--
	if c > 0 {
		s.refCount[id] = c
		return
	}

	if ok {
		delete(s.refCount, id)
	}

	if lock, ok := s.locks[id]; ok {
		_ = lock.Close()
		delete(s.locks, id)
	}
}

type localSess struct {
	cfg    *SessionConfig
	srv    *localSrv
	closed bool
}

func (s *localSess) NewLocker(id string) Locker {
	return &localLocker{id: id, sess: s}
}

func (s *localSess) Close() error {
	if s.closed {
		return ErrAlreadyClosed.New()
	}

	s.closed = true
	return nil
}

type localLocker struct {
	id     string
	sess   *localSess
	lock   *localLock
	unlock chan struct{}
}

func (l *localLocker) Lock() (<-chan struct{}, error) {
	lock := l.sess.srv.getLock(l.id)
	ok := lock.Lock(l.sess.cfg.Timeout)
	if !ok {
		l.sess.srv.freeLock(l.id)
		return nil, ErrCanceled.New()
	}

	l.lock = lock
	l.unlock = make(chan struct{})
	return l.unlock, nil
}

func (l *localLocker) Unlock() error {
	lock := l.lock
	if lock == nil {
		return nil
	}

	l.lock = nil
	close(l.unlock)
	lock.Unlock()
	l.sess.srv.freeLock(l.id)
	return nil
}
