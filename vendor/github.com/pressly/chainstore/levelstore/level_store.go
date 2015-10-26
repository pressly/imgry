package levelstore

import (
	"os"

	"github.com/syndtr/goleveldb/leveldb"
)

type levelStore struct {
	storePath string
	db        *leveldb.DB
	opened    bool
}

func New(storePath string) *levelStore {
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

func (s *levelStore) Put(key string, val []byte) error {
	return s.db.Put([]byte(key), val, nil)
}

func (s *levelStore) Get(key string) (val []byte, err error) {
	val, err = s.db.Get([]byte(key), nil)
	if err != nil && err != leveldb.ErrNotFound {
		return nil, err
	}
	return val, nil
}

func (s *levelStore) Del(key string) error {
	return s.db.Delete([]byte(key), nil)
}
