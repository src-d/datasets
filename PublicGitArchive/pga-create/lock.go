package indexer

import (
	"sync"
)

type locker struct {
	mut   sync.Mutex
	locks map[string]*sync.Mutex
}

func newLocker() *locker {
	return &locker{
		locks: make(map[string]*sync.Mutex),
	}
}

func (l *locker) lock(id string) *sync.Mutex {
	var lock *sync.Mutex

	l.mut.Lock()
	defer l.mut.Unlock()
	if lo, ok := l.locks[id]; !ok {
		lock = new(sync.Mutex)
		l.locks[id] = lock
	} else {
		lock = lo
	}

	return lock
}
