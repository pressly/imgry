package mockstore

import (
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/pressly/chainstore"
	"golang.org/x/net/context"
)

var _ = chainstore.Store(&mockStore{})

type mockStore struct {
	mu      sync.RWMutex
	data    map[string][]byte
	cfg     *Config
	timer   *time.Timer
	timerMu sync.Mutex
	closed  bool
}

// Config stores settings for this store.
type Config struct {
	Capacity    int64
	SuccessRate float32
	Delay       time.Duration
	Timeout     time.Duration
}

// New creates and returns a mock chainstore.Store.
func New(cfg *Config) chainstore.Store {
	if cfg == nil {
		cfg = &Config{
			Capacity:    1000,
			SuccessRate: 1.0,
			Delay:       0,
		}
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = chainstore.DefaultTimeout
	}
	mockStore := &mockStore{
		data: make(map[string][]byte, cfg.Capacity),
		cfg:  cfg,
	}
	return mockStore
}

func (s *mockStore) success() bool {
	return rand.Float32() < s.cfg.SuccessRate
}

func (s *mockStore) Open() error {
	if !s.success() {
		return errors.New("Failed to open: random fail.")
	}
	return nil
}

func (s *mockStore) Close() error {
	if !s.success() {
		return errors.New("Failed to close: random fail.")
	}

	s.timerMu.Lock()
	s.timer.Reset(0)
	s.timerMu.Unlock()

	s.mu.Lock()
	s.closed = true
	s.data = nil
	s.mu.Unlock()

	return nil
}

func (s *mockStore) delay() error {
	s.timerMu.Lock()
	s.timer = time.NewTimer(s.cfg.Delay)
	s.timerMu.Unlock()

	timeout := time.NewTimer(s.cfg.Timeout)

	select {
	case <-s.timer.C:
		return nil
	case <-timeout.C:
		return chainstore.ErrTimeout
	}
}

func (s *mockStore) doGet(key string) ([]byte, error) {
	if err := s.delay(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, errors.New("Store is closed.")
	}

	if v, ok := s.data[key]; ok {
		return v, nil
	}

	return nil, errors.New("No such key.")
}

func (s *mockStore) doDel(key string) error {
	if err := s.delay(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return errors.New("Store is closed.")
	}

	if _, ok := s.data[key]; ok {
		delete(s.data, key)
	}

	return nil
}

func (s *mockStore) doPut(key string, val []byte) error {
	if err := s.delay(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return errors.New("Store is closed.")
	}

	s.data[key] = val
	return nil
}

func (s *mockStore) Put(ctx context.Context, key string, val []byte) error {
	if !s.success() {
		return errors.New("Failed to put key in store: random fail.")
	}

	errCh := make(chan error)

	go func() {
		errCh <- s.doPut(key, val)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}

	panic("reached")
}

func (s *mockStore) Get(ctx context.Context, key string) ([]byte, error) {
	var v []byte

	if !s.success() {
		return nil, errors.New("Failed to get key from store: random fail.")
	}

	errCh := make(chan error)

	go func() {
		var err error
		v, err = s.doGet(key)
		errCh <- err
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-errCh:
		return v, err
	}

	panic("reached")
}

func (s *mockStore) Del(ctx context.Context, key string) error {
	if !s.success() {
		return errors.New("Failed to delete key from store: random fail.")
	}

	errCh := make(chan error)

	go func() {
		errCh <- s.doDel(key)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}

	panic("reached")
}
