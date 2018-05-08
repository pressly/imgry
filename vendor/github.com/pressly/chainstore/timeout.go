package chainstore

import (
	"context"
	"time"
)

func Timeout(d time.Duration, stores ...Store) Store {
	return &timeoutManager{
		timeout: d,
		chain:   New(stores...),
	}
}

type timeoutManager struct {
	timeout time.Duration
	chain   Store
}

func (s *timeoutManager) Open() (err error)  { return s.chain.Open() }
func (s *timeoutManager) Close() (err error) { return s.chain.Close() }

func (s *timeoutManager) Put(ctx context.Context, key string, val []byte) (err error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	return s.chain.Put(ctx, key, val)
}

func (s *timeoutManager) Get(ctx context.Context, key string) (data []byte, err error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	return s.chain.Get(ctx, key)
}

func (s *timeoutManager) Del(ctx context.Context, key string) (err error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	return s.chain.Del(ctx, key)

}
