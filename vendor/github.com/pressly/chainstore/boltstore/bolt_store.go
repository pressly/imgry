package boltstore

import (
	"os"
	"path/filepath"
	"time"

	"github.com/boltdb/bolt"
)

type boltStore struct {
	storePath  string
	bucketName []byte

	db     *bolt.DB
	bucket *bolt.Bucket
	opened bool
}

func New(storePath string, bucketName string) *boltStore {
	return &boltStore{storePath: storePath, bucketName: []byte(bucketName)}
}

func (s *boltStore) Open() (err error) {
	if s.opened {
		return
	}

	// Create the store directory if doesnt exist
	storeDir := filepath.Dir(s.storePath)
	if _, err = os.Stat(storeDir); os.IsNotExist(err) {
		err = os.MkdirAll(storeDir, 0755)
		if err != nil {
			return
		}
	}

	s.db, err = bolt.Open(s.storePath, 0660, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return
	}

	// Initialize all required buckets
	err = s.db.Update(func(tx *bolt.Tx) (err error) {
		s.bucket, err = tx.CreateBucketIfNotExists(s.bucketName)
		return err
	})
	if err == nil {
		s.opened = true
	}
	return
}

func (s *boltStore) Close() (err error) {
	err = s.db.Close()
	if err == nil {
		s.opened = false
	}
	return
}

func (s *boltStore) Put(key string, val []byte) (err error) {
	err = s.db.Batch(func(tx *bolt.Tx) error {
		b := tx.Bucket(s.bucketName)
		return b.Put([]byte(key), val)
	})
	return
}

func (s *boltStore) Get(key string) (val []byte, err error) {
	err = s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(s.bucketName)
		val = b.Get([]byte(key))
		return nil
	})
	return
}

func (s *boltStore) Del(key string) (err error) {
	err = s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(s.bucketName)
		return b.Delete([]byte(key))
	})
	return
}
