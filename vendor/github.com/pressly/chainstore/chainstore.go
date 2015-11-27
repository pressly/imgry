package chainstore

import (
	"regexp"

	"golang.org/x/net/context"
)

var (
	keyInvalidator = regexp.MustCompile(`(i?)[^a-z0-9\/_\-:\.]`)
)

const (
	maxKeyLen = 256
)

// Store represents a store than can be used as a chainstore link.
type Store interface {
	Open() error
	Close() error
	Put(ctx context.Context, key string, val []byte) error
	Get(ctx context.Context, key string) ([]byte, error)
	Del(ctx context.Context, key string) error
}

// Chain represents a store chain.
type Chain struct {
	stores      []Store
	async       bool
	errCallback func(error)
}

// New creates a new store chain backed by the passed stores.
func New(stores ...Store) Store {
	return &Chain{stores, false, nil}
}

// Async creates and async store.
func Async(errCallback func(error), stores ...Store) Store {
	return &Chain{stores, true, errCallback}
}

// Open all the stores.
func (c *Chain) Open() (err error) {
	for _, s := range c.stores {
		err = s.Open()
		if err != nil {
			return // return first error that comes up
		}
	}
	return
}

// Close closes all the stores.
func (c *Chain) Close() error {
	errs := fewerrors{}
	for _, s := range c.stores {
		err := s.Close()
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

// Put propagates a key-value pair to all stores.
func (c *Chain) Put(ctx context.Context, key string, val []byte) (err error) {
	if !IsValidKey(key) {
		return ErrInvalidKey
	}

	fn := func() (err error) {
		for _, s := range c.stores {
			err = s.Put(ctx, key, val)
			if err != nil {
				if c.errCallback != nil {
					c.errCallback(err)
				}
				return
			}
		}
		return
	}
	if c.async {
		go fn()
	} else {
		err = fn()
	}
	return
}

// Get returns the value identified by the given key. This is a sequential
// scan. When a value is found it gets propagated to all the stores that do not
// have it.
func (c *Chain) Get(ctx context.Context, key string) (val []byte, err error) {
	if !IsValidKey(key) {
		return nil, ErrInvalidKey
	}

	for i, s := range c.stores {
		val, err = s.Get(ctx, key)
		if err != nil {
			if c.errCallback != nil {
				c.errCallback(err)
			}
			return
		}

		if len(val) > 0 {
			if i > 0 {
				// put the value in all other stores up the chain
				fn := func() {
					for n := i - 1; n >= 0; n-- {
						err := c.stores[n].Put(ctx, key, val)
						if c.errCallback != nil {
							c.errCallback(err)
						}
					}
				}
				go fn()
			}

			// return the first value found on the chain
			return
		}
	}
	return
}

// Del removes a key from all stores.
func (c *Chain) Del(ctx context.Context, key string) (err error) {
	if !IsValidKey(key) {
		return ErrInvalidKey
	}

	fn := func() (err error) {
		for _, s := range c.stores {
			err = s.Del(ctx, key)
			if err != nil {
				if c.errCallback != nil {
					c.errCallback(err)
				}
				return
			}
		}
		return
	}
	if c.async {
		go fn()
	} else {
		err = fn()
	}
	return
}

func IsValidKey(key string) bool {
	return len(key) <= maxKeyLen && !keyInvalidator.MatchString(key)
}
