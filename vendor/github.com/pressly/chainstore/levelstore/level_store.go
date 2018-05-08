package levelstore

import (
	"os"

	"github.com/pressly/chainstore"
	"github.com/syndtr/goleveldb/leveldb"
	"golang.org/x/net/context"
)

type levelStore struct {
	storePath string
	db        *leveldb.DB
	opened    bool
}

// New returns returns a leveldb backed store.
func New(storePath string) chainstore.Store {
	return &levelStore{storePath: storePath}
}

func (s *levelStore) Open() (err error) {
	if s.opened {
		return
	}

	// Create the store directory if doesnt exist
	if _, err = os.Stat(s.storePath); os.IsNotExist(err) {
		err = os.MkdirAll(s.storePath, 0755)
		if err != nil {
			return
		}
	}

	s.db, err = leveldb.OpenFile(s.storePath, nil)
	if err == nil {
		s.opened = true
	}
	return
}

func (s *levelStore) Close() (err error) {
	err = s.db.Close()
	if err == nil {
		s.opened = false
	}
	return
}

func (s *levelStore) Put(ctx context.Context, key string, val []byte) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return s.db.Put([]byte(key), val, nil)
	}
}

func (s *levelStore) Get(ctx context.Context, key string) (val []byte, err error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		val, err = s.db.Get([]byte(key), nil)
		if err != nil && err != leveldb.ErrNotFound {
			return nil, err
		}
		return val, nil
	}
}

func (s *levelStore) Del(ctx context.Context, key string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return s.db.Delete([]byte(key), nil)
	}
}
