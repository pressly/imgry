package boltstore

import (
	"os"
	"path/filepath"

	"github.com/boltdb/bolt"
	"github.com/pressly/chainstore"
	"golang.org/x/net/context"
)

type boltStore struct {
	storePath  string
	bucketName []byte

	db     *bolt.DB
	bucket *bolt.Bucket
	opened bool
}

// New creates and returns a boltdb based store.
func New(storePath string, bucketName string) chainstore.Store {
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

	s.db, err = bolt.Open(s.storePath, 0660, nil)
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

func (s *boltStore) Put(ctx context.Context, key string, val []byte) (err error) {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		err = s.db.Batch(func(tx *bolt.Tx) error {
			b := tx.Bucket(s.bucketName)
			return b.Put([]byte(key), val)
		})
		return
	}
}

func (s *boltStore) Get(ctx context.Context, key string) (val []byte, err error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		err = s.db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket(s.bucketName)
			raw := b.Get([]byte(key))
			val = make([]byte, len(raw))
			copy(val, raw)
			return nil
		})
		return
	}
}

func (s *boltStore) Del(ctx context.Context, key string) (err error) {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		err = s.db.Update(func(tx *bolt.Tx) error {
			b := tx.Bucket(s.bucketName)
			return b.Delete([]byte(key))
		})
		return
	}
}
