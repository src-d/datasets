package queue

import (
	"fmt"
	"time"

	"gopkg.in/src-d/go-errors.v1"

	"github.com/satori/go.uuid"
	"gopkg.in/vmihailenco/msgpack.v2"
)

type contentType string

const msgpackContentType contentType = "application/msgpack"

// Job contains the information for a job to be published to a queue.
type Job struct {
	// ID of the job.
	ID string
	// Priority is the priority level.
	Priority Priority
	// Timestamp is the time of creation.
	Timestamp time.Time
	// Retries is the number of times this job can be processed before being rejected.
	Retries int32
	// ErrorType is the kind of error that made the job failing.
	ErrorType string

	contentType  contentType
	raw          []byte
	acknowledger Acknowledger
	tag          uint64
}

// Acknowledger represents the object in charge of acknowledgement
// management for a job.  When a job is acknowledged using any of the
// functions in this interface, it will be considered delivered by the
// queue.
type Acknowledger interface {
	// Ack is called when the Job has finished.
	Ack() error
	// Reject is called if the job has errored. The parameter indicates
	// whether the job should be put back in the queue or not.
	Reject(requeue bool) error
}

// NewJob creates a new Job with default values, a new unique ID and current
// timestamp.
func NewJob() (*Job, error) {
	u, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	return &Job{
		ID:          u.String(),
		Priority:    PriorityNormal,
		Timestamp:   time.Now(),
		contentType: msgpackContentType,
	}, nil
}

// SetPriority sets job priority
func (j *Job) SetPriority(priority Priority) {
	j.Priority = priority
}

// Encode encodes the payload to the wire format used.
func (j *Job) Encode(payload interface{}) error {
	var err error
	j.raw, err = encode(msgpackContentType, &payload)
	if err != nil {
		return err
	}

	return nil
}

// Decode decodes the payload from the wire format.
func (j *Job) Decode(payload interface{}) error {
	return decode(msgpackContentType, j.raw, &payload)
}

var ErrCantAck = errors.NewKind("can't acknowledge this message, it does not come from a queue")

// Ack is called when the job is finished.
func (j *Job) Ack() error {
	if j.acknowledger == nil {
		return ErrCantAck.New()
	}
	return j.acknowledger.Ack()
}

// Reject is called when the job errors. The parameter is true if and only if the
// job should be put back in the queue.
func (j *Job) Reject(requeue bool) error {
	if j.acknowledger == nil {
		return ErrCantAck.New()
	}
	return j.acknowledger.Reject(requeue)
}

func encode(mime contentType, p interface{}) ([]byte, error) {
	switch mime {
	case msgpackContentType:
		return msgpack.Marshal(p)
	default:
		return nil, fmt.Errorf("unknown content type: %s", mime)
	}
}

func decode(mime contentType, r []byte, p interface{}) error {
	switch mime {
	case msgpackContentType:
		return msgpack.Unmarshal(r, p)
	default:
		return fmt.Errorf("unknown content type: %s", mime)
	}
}
