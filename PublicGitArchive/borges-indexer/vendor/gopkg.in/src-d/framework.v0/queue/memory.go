package queue

import (
	"sync"
	"time"
)

type memoryBroker struct {
	queues map[string]Queue
}

// Creates a new Broker for an in-memory queue.
func NewMemoryBroker() Broker {
	return &memoryBroker{make(map[string]Queue)}
}

// Queue returns the queue with the given name.
func (b *memoryBroker) Queue(name string) (Queue, error) {
	if _, ok := b.queues[name]; !ok {
		b.queues[name] = &memoryQueue{jobs: make([]*Job, 0, 10)}
	}

	return b.queues[name], nil
}

// Close closes the connection in the Broker.
func (b *memoryBroker) Close() error {
	return nil
}

type memoryQueue struct {
	jobs       []*Job
	buriedJobs []*Job
	sync.RWMutex
	idx                int
	publishImmediately bool
}

// Publish publishes a Job to the queue.
func (q *memoryQueue) Publish(j *Job) error {
	if j == nil || len(j.raw) == 0 {
		return ErrEmptyJob
	}

	q.Lock()
	defer q.Unlock()
	q.jobs = append(q.jobs, j)
	return nil
}

// PublishDelayed publishes a Job to the queue with a given delay.
func (q *memoryQueue) PublishDelayed(j *Job, delay time.Duration) error {
	if j == nil || len(j.raw) == 0 {
		return ErrEmptyJob
	}

	if q.publishImmediately {
		return q.Publish(j)
	}
	go func() {
		<-time.After(delay)
		q.Publish(j)
	}()
	return nil
}

func (q *memoryQueue) RepublishBuried() error {
	for _, j := range q.buriedJobs {
		q.Publish(j)
	}
	return nil
}

// Transaction calls the given callback inside a transaction.
func (q *memoryQueue) Transaction(txcb TxCallback) error {
	txQ := &memoryQueue{jobs: make([]*Job, 0, 10), publishImmediately: true}
	if err := txcb(txQ); err != nil {
		return err
	}

	q.jobs = append(q.jobs, txQ.jobs...)
	return nil
}

// Consume implements Queue.  MemoryQueues have infinite advertised window.
func (q *memoryQueue) Consume(_ int) (JobIter, error) {
	return &memoryJobIter{q: q, RWMutex: &q.RWMutex}, nil
}

type memoryJobIter struct {
	q      *memoryQueue
	closed bool
	*sync.RWMutex
}

type memoryAck struct {
	q *memoryQueue
	j *Job
}

// Ack is called when the Job has finished.
func (*memoryAck) Ack() error {
	return nil
}

// Reject is called when the Job has errored. The argument indicates whether the Job
// should be put back in queue or not.  If requeue is false, the job will go to the buried
// queue until Queue.RepublishBuried() is called.
func (a *memoryAck) Reject(requeue bool) error {
	if !requeue {
		// Send to the buried queue for later republishing
		a.q.buriedJobs = append(a.q.buriedJobs, a.j)
		return nil
	}

	return a.q.Publish(a.j)
}

func (i *memoryJobIter) isClosed() bool {
	i.RLock()
	defer i.RUnlock()
	return i.closed
}

// Next returns the next job in the iter.
func (i *memoryJobIter) Next() (*Job, error) {
	for {
		if i.isClosed() {
			return nil, ErrAlreadyClosed
		}

		j, err := i.next()
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		return j, nil
	}
}

func (i *memoryJobIter) next() (*Job, error) {
	i.Lock()
	defer i.Unlock()
	if len(i.q.jobs) <= i.q.idx {
		return nil, nil
	}
	j := i.q.jobs[i.q.idx]
	i.q.idx++
	j.tag = 1
	j.acknowledger = &memoryAck{j: j, q: i.q}
	return j, nil
}

// Close closes the iter.
func (i *memoryJobIter) Close() error {
	i.Lock()
	defer i.Unlock()
	i.closed = true
	return nil
}
