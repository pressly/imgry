package chainstore

import (
	"errors"
	"io/ioutil"
	"regexp"
)

var (
	ErrInvalidKey = errors.New("Invalid key")

	KeyInvalidator = regexp.MustCompile(`(i?)[^a-z0-9\/_\-:\.]`)
)

const (
	MaxKeyLen = 256
)

type Store interface {
	Open() error
	Close() error
	Put(key string, val []byte) error
	Get(key string) ([]byte, error)
	Del(key string) error
}

// TODO: how can we check if a store has been opened...?

type Chain struct {
	stores []Store
	async  bool
}

func New(stores ...Store) Store {
	return &Chain{stores, false}
	// TODO: make the chain..
	// call Open(), but in case of error..?
}

func Async(stores ...Store) Store {
	return &Chain{stores, true}
}

func (c *Chain) Open() (err error) {
	for _, s := range c.stores {
		err = s.Open()
		if err != nil {
			return // return first error that comes up
		}
	}
	return
}

func (c *Chain) Close() (err error) {
	for _, s := range c.stores {
		err = s.Close()
		// TODO: we shouldn't stop on first error.. should keep trying to close
		// and record errors separately
		if err != nil {
			return
		}
	}
	return
}

func (c *Chain) Put(key string, val []byte) (err error) {
	if !IsValidKey(key) {
		return ErrInvalidKey
	}

	fn := func() (err error) {
		for _, s := range c.stores {
			err = s.Put(key, val)
			if err != nil {
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

func (c *Chain) Get(key string) (val []byte, err error) {
	if !IsValidKey(key) {
		return nil, ErrInvalidKey
	}

	for i, s := range c.stores {
		val, err = s.Get(key)
		if err != nil {
			return
		}

		if len(val) > 0 {
			if i > 0 {
				// put the value in all other stores up the chain
				fn := func() {
					for n := i - 1; n >= 0; n-- {
						c.stores[n].Put(key, val) // errors..?
					}
				}
				// if c.async { } else { } ....?
				go fn()
			}

			// return the first value found on the chain
			return
		}
	}
	return
}

func (c *Chain) Del(key string) (err error) {
	if !IsValidKey(key) {
		return ErrInvalidKey
	}

	fn := func() (err error) {
		for _, s := range c.stores {
			err = s.Del(key)
			if err != nil {
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
	return len(key) <= MaxKeyLen && !KeyInvalidator.MatchString(key)
}

func TempDir() string {
	path, _ := ioutil.TempDir("", "chainstore-")
	return path
}
