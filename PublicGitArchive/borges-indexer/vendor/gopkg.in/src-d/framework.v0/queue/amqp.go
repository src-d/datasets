package queue

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/streadway/amqp"
	log15 "gopkg.in/inconshreveable/log15.v2"
)

var consumerSeq uint64

const buriedQueueSuffix = ".buriedQueue"
const buriedQueueExchangeSuffix = ".buriedExchange"
const buriedNonBlockingRetries = 3

// AMQPBroker implements the Broker interface for AMQP.
type AMQPBroker struct {
	mut        sync.RWMutex
	conn       *amqp.Connection
	ch         *amqp.Channel
	connErrors chan *amqp.Error
	stop       chan struct{}
}

type connection interface {
	connection() *amqp.Connection
	channel() *amqp.Channel
}

// NewAMQPBroker creates a new AMQPBroker.
func NewAMQPBroker(url string) (Broker, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %s", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open a channel: %s", err)
	}

	b := &AMQPBroker{
		conn: conn,
		ch:   ch,
		stop: make(chan struct{}),
	}

	go b.manageConnection(url)

	return b, nil
}

func connect(url string) (*amqp.Connection, *amqp.Channel) {
	// first try to connect again
	var conn *amqp.Connection
	var err error
	for {
		conn, err = amqp.Dial(url)
		if err != nil {
			log15.Error("error connecting to amqp", "err", err)
			<-time.After(1 * time.Second)
			continue
		}

		break
	}

	// try to get the channel again
	var ch *amqp.Channel
	for {
		ch, err = conn.Channel()
		if err != nil {
			log15.Error("error creatting channel", "err", err)
			<-time.After(1 * time.Second)
			continue
		}

		break
	}

	return conn, ch
}

func (b *AMQPBroker) manageConnection(url string) {
	b.connErrors = make(chan *amqp.Error)
	b.conn.NotifyClose(b.connErrors)

	for {
		select {
		case err := <-b.connErrors:
			log15.Error("amqp connection error", "err", err)
			b.mut.Lock()
			if err != nil {
				b.conn, b.ch = connect(url)

				b.connErrors = make(chan *amqp.Error)
				b.conn.NotifyClose(b.connErrors)
			}

			b.mut.Unlock()
		case <-b.stop:
			return
		}
	}
}

func (b *AMQPBroker) connection() *amqp.Connection {
	b.mut.Lock()
	defer b.mut.Unlock()
	return b.conn
}

func (b *AMQPBroker) channel() *amqp.Channel {
	b.mut.Lock()
	defer b.mut.Unlock()
	return b.ch
}

func (b *AMQPBroker) newBuriedQueue(mainQueueName string) (q amqp.Queue, rex string, err error) {
	ch, err := b.conn.Channel()
	if err != nil {
		return
	}

	buriedName := mainQueueName + buriedQueueSuffix
	rex = mainQueueName + buriedQueueExchangeSuffix

	if err = ch.ExchangeDeclare(rex, "fanout", true, false, false, false, nil); err != nil {
		return
	}

	q, err = b.ch.QueueDeclare(
		buriedName,
		true,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		return
	}

	if err = ch.QueueBind(buriedName, "", rex, true, nil); err != nil {
		return
	}

	return
}

// Queue returns the queue with the given name.
func (b *AMQPBroker) Queue(name string) (Queue, error) {
	buriedQueue, rex, err := b.newBuriedQueue(name)
	if err != nil {
		return nil, err
	}

	q, err := b.ch.QueueDeclare(
		name,  // name
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		amqp.Table{
			"x-dead-letter-exchange":    rex,
			"x-dead-letter-routing-key": name,
		},
	)

	if err != nil {
		return nil, err
	}

	return &AMQPQueue{
		conn:        b,
		queue:       q,
		buriedQueue: &AMQPQueue{conn: b, queue: buriedQueue},
	}, nil
}

// Close closes all the connections managed by the broker.
func (b *AMQPBroker) Close() error {
	close(b.stop)

	if err := b.channel().Close(); err != nil {
		return err
	}

	if err := b.connection().Close(); err != nil {
		return err
	}

	return nil
}

// AMQPQueue implements the Queue interface for the AMQP.
type AMQPQueue struct {
	conn        connection
	queue       amqp.Queue
	buriedQueue *AMQPQueue
}

// Publish publishes the given Job to the Queue.
func (q *AMQPQueue) Publish(j *Job) error {
	if j == nil || len(j.raw) == 0 {
		return ErrEmptyJob
	}

	return q.conn.channel().Publish(
		"",           // exchange
		q.queue.Name, // routing key
		false,        // mandatory
		false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			MessageId:    j.ID,
			Priority:     uint8(j.Priority),
			Timestamp:    j.Timestamp,
			ContentType:  string(j.contentType),
			Body:         j.raw,
		},
	)
}

// PublishDelayed publishes the given Job with a given delay. Delayed messages
// wont go into the buried queue if they fail.
func (q *AMQPQueue) PublishDelayed(j *Job, delay time.Duration) error {
	if j == nil || len(j.raw) == 0 {
		return ErrEmptyJob
	}

	ttl := delay / time.Millisecond
	delayedQueue, err := q.conn.channel().QueueDeclare(
		j.ID,  // name
		true,  // durable
		true,  // delete when unused
		false, // exclusive
		false, // no-wait
		amqp.Table{
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": q.queue.Name,
			"x-message-ttl":             int64(ttl),
			"x-expires":                 int64(ttl) * 2,
		},
	)
	if err != nil {
		return err
	}

	return q.conn.channel().Publish(
		"", // exchange
		delayedQueue.Name,
		false,
		false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			MessageId:    j.ID,
			Priority:     uint8(j.Priority),
			Timestamp:    j.Timestamp,
			ContentType:  string(j.contentType),
			Body:         j.raw,
		},
	)
}

// RepublishBuried will republish in the main queue all the jobs that timed out without Ack
// or were Rejected with requeue = False.
func (q *AMQPQueue) RepublishBuried() error {
	var buriedJobs []*Job
	err := q.getBuriedJobs(&buriedJobs)
	if err != nil {
		return err
	}

	for _, j := range buriedJobs {
		if err = q.Publish(j); err != nil {
			return err
		}
	}
	return nil
}

func (q *AMQPQueue) getBuriedJobs(jobs *[]*Job) error {
	if q.buriedQueue == nil {
		return fmt.Errorf("buriedQueue is nil, called RepublishBuried on the internal buried queue?")
	}

	// enforce prefetching only one job
	iter, err := q.buriedQueue.Consume(1)
	if err != nil {
		return err
	}

	defer iter.Close()

	retries := 0
	for {
		j, err := iter.(*AMQPJobIter).nextNonBlocking()
		if err != nil {
			return err
		}

		if j == nil {
			// check (in non blocking mode) up to "buriedNonBlockingRetries" with
			// a small delay between them just in case some job is arriving, return
			// if there is nothing after all the retries (meaning: BuriedQueue is surely
			// empty or any arriving jobs will have to wait to the next call).
			if retries > buriedNonBlockingRetries {
				return nil
			}

			time.Sleep(50 * time.Millisecond)
			retries++
			continue
		}

		if err = j.Ack(); err != nil {
			return err
		}

		retries = 0
		*jobs = append(*jobs, j)
	}
}

// Transaction executes the given callback inside a transaction.
func (q *AMQPQueue) Transaction(txcb TxCallback) error {
	ch, err := q.conn.connection().Channel()
	if err != nil {
		return fmt.Errorf("failed to open a channel: %s", err)
	}

	defer ch.Close()

	if err := ch.Tx(); err != nil {
		return err
	}

	txQueue := &AMQPQueue{
		conn: &AMQPBroker{
			conn: q.conn.connection(),
			ch:   ch,
		},
		queue: q.queue,
	}

	err = txcb(txQueue)
	if err != nil {
		if err := ch.TxRollback(); err != nil {
			return err
		}

		return err
	}

	if err := ch.TxCommit(); err != nil {
		return err
	}

	return nil
}

// Implements Queue.  The advertisedWindow value will be the exact
// number of undelivered jobs in transit, not just the minium.
func (q *AMQPQueue) Consume(advertisedWindow int) (JobIter, error) {
	ch, err := q.conn.connection().Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open a channel: %s", err)
	}

	// enforce prefetching only one job, if this is removed the whole queue
	// will be consumed.
	if err := ch.Qos(advertisedWindow, 0, false); err != nil {
		return nil, err
	}

	id := q.consumeID()
	c, err := ch.Consume(
		q.queue.Name, // queue
		id,           // consumer
		false,        // autoAck
		false,        // exclusive
		false,        // noLocal
		false,        // noWait
		nil,          // args
	)
	if err != nil {
		return nil, err
	}

	return &AMQPJobIter{id: id, ch: ch, c: c}, nil
}

func (q *AMQPQueue) consumeID() string {
	return fmt.Sprintf("%s-%s-%d",
		os.Args[0],
		q.queue.Name,
		atomic.AddUint64(&consumerSeq, 1),
	)
}

// AMQPJobIter implements the JobIter interface for AMQP.
type AMQPJobIter struct {
	id string
	ch *amqp.Channel
	c  <-chan amqp.Delivery
}

// Next returns the next job in the iter.
func (i *AMQPJobIter) Next() (*Job, error) {
	d, ok := <-i.c
	if !ok {
		return nil, ErrAlreadyClosed
	}

	return fromDelivery(&d)
}

func (i *AMQPJobIter) nextNonBlocking() (*Job, error) {
	select {
	case d, ok := <-i.c:
		if !ok {
			return nil, ErrAlreadyClosed
		}

		return fromDelivery(&d)
	default:
		return nil, nil
	}
}

// Close closes the channel of the JobIter.
func (i *AMQPJobIter) Close() error {
	if err := i.ch.Cancel(i.id, false); err != nil {
		return err
	}

	return i.ch.Close()
}

// AMQPAcknowledger implements the Acknowledger for AMQP.
type AMQPAcknowledger struct {
	ack amqp.Acknowledger
	id  uint64
}

// Ack signals ackwoledgement.
func (a *AMQPAcknowledger) Ack() error {
	return a.ack.Ack(a.id, false)
}

// Reject signals rejection. If requeue is false, the job will go to the buried
// queue until Queue.RepublishBuried() is called.
func (a *AMQPAcknowledger) Reject(requeue bool) error {
	return a.ack.Reject(a.id, requeue)
}

func fromDelivery(d *amqp.Delivery) (*Job, error) {
	j, err := NewJob()
	if err != nil {
		return nil, err
	}

	j.ID = d.MessageId
	j.Priority = Priority(d.Priority)
	j.Timestamp = d.Timestamp
	j.contentType = contentType(d.ContentType)
	j.acknowledger = &AMQPAcknowledger{d.Acknowledger, d.DeliveryTag}
	j.tag = d.DeliveryTag
	j.raw = d.Body

	return j, nil
}
