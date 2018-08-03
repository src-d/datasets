package lock

import (
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/clientv3/concurrency"
	"golang.org/x/net/context"
)

const ServiceEtcd = "etcd"

func init() {
	Services[ServiceEtcd] = NewEtcd
}

// NewEtcd creates a new locking service based on etcd given a connection string.
// The connection string has the following form:
//
//   etcd:<endpoints>[?<opt1>=<val1>&<opt2>=<val2>]
//
// For example:
//
//   etcd:http//foo:8888,http://bar:9999?dial-timeout=2s&reject-old-cluster=true
//
// Valid options are:
//
//   - auto-sync-interval (time duration)
//   - dial-timeout (time duration)
//   - dial-keep-alive-time (time duration)
//   - dial-keep-alive-timeout (time duration)
//   - username (string)
//   - password (string)
//   - reject-old-cluster (boolean)
//
// For further information about each option, check the etcd godocs at:
// https://godoc.org/github.com/coreos/etcd/clientv3#Config
func NewEtcd(connstr string) (Service, error) {
	cfg, err := parseEtcdConnectionstring(connstr)
	if err != nil {
		return nil, err
	}

	return &etcdSrv{
		cfg: cfg,
		m:   &sync.Mutex{},
	}, nil
}

func parseEtcdConnectionstring(connstr string) (cfg clientv3.Config, err error) {
	u, err := url.Parse(connstr)
	if err != nil {
		return cfg, ErrInvalidConnectionString.Wrap(err, "invalid URL")
	}

	if u.Scheme != ServiceEtcd {
		return cfg, ErrUnsupportedService.New(u.Scheme)
	}

	if u.Opaque == "" {
		return cfg, ErrInvalidConnectionString.New("URI must be opaque")
	}

	cfg.Endpoints = strings.Split(u.Opaque, ",")
	for key, vals := range u.Query() {
		val := vals[len(vals)-1]
		switch key {
		case "auto-sync-interval":
			cfg.AutoSyncInterval, err = time.ParseDuration(val)
			if err != nil {
				return cfg, ErrInvalidConnectionString.Wrap(err, key)
			}
		case "dial-timeout":
			cfg.DialTimeout, err = time.ParseDuration(val)
			if err != nil {
				return cfg, ErrInvalidConnectionString.Wrap(err, key)
			}
		case "dial-keep-alive-time":
			cfg.DialKeepAliveTime, err = time.ParseDuration(val)
			if err != nil {
				return cfg, ErrInvalidConnectionString.Wrap(err, key)
			}
		case "dial-keep-alive-timeout":
			cfg.DialKeepAliveTimeout, err = time.ParseDuration(val)
			if err != nil {
				return cfg, ErrInvalidConnectionString.Wrap(err, key)
			}
		case "username":
			cfg.Username = val
		case "password":
			cfg.Password = val
		case "reject-old-cluster":
			cfg.RejectOldCluster, err = strconv.ParseBool(val)
			if err != nil {
				return cfg, ErrInvalidConnectionString.Wrap(err, key)
			}
		default:
			return cfg, ErrInvalidConnectionString.New(fmt.Sprintf("invalid option: %s", key))
		}
	}

	return cfg, nil
}

type etcdSrv struct {
	cfg    clientv3.Config
	client *clientv3.Client
	m      *sync.Mutex
	closed bool
}

func (s *etcdSrv) connect() error {
	s.m.Lock()
	defer s.m.Unlock()

	if s.client == nil {
		client, err := clientv3.New(s.cfg)
		if err != nil {
			return err
		}

		s.client = client
	}

	return nil
}

func (s *etcdSrv) NewSession(cfg *SessionConfig) (Session, error) {
	if err := s.connect(); err != nil {
		return nil, err
	}

	ttl := int(math.Ceil(cfg.TTL.Seconds()))
	session, err := concurrency.NewSession(s.client, concurrency.WithTTL(ttl))
	if err != nil {
		return nil, err
	}

	return &etcdSess{
		cfg:     cfg,
		session: session,
	}, nil
}

func (l *etcdSrv) Close() error {
	if l.closed {
		return ErrAlreadyClosed.New()
	}

	defer func() { l.closed = true }()

	if l.client == nil {
		return nil
	}

	return l.client.Close()
}

type etcdSess struct {
	cfg     *SessionConfig
	session *concurrency.Session
}

func (s *etcdSess) NewLocker(id string) Locker {
	return &etcdLock{
		cfg:     s.cfg,
		session: s.session,
		mutex:   concurrency.NewMutex(s.session, id),
	}
}

func (s *etcdSess) Close() error {
	session := s.session
	if session == nil {
		return ErrAlreadyClosed.New()
	}

	s.session = nil
	return session.Close()
}

type etcdLock struct {
	cfg     *SessionConfig
	session *concurrency.Session
	mutex   *concurrency.Mutex
}

func (l *etcdLock) Lock() (<-chan struct{}, error) {
	ctx := context.Background()

	timeout := l.cfg.Timeout
	if timeout > 0 {
		if timeout < time.Second {
			timeout = time.Second
		}

		ctx, _ = context.WithTimeout(ctx, timeout)
	}

	if err := l.mutex.Lock(ctx); err != nil {
		if isContextDeadlineExceededError(err) {
			return nil, ErrCanceled.Wrap(err)
		}

		return nil, err
	}

	return l.session.Done(), nil
}

func (m *etcdLock) Unlock() error {
	return m.mutex.Unlock(context.TODO())
}

func isContextDeadlineExceededError(err error) bool {
	return strings.Contains(err.Error(), "context deadline exceeded")
}
