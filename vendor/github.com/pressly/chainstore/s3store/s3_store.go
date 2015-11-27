package s3store

import (
	"net/http"

	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"
	"github.com/pressly/chainstore"
	"golang.org/x/net/context"
)

type s3Store struct {
	BucketID, AccessKey, SecretKey string

	conn   *s3.S3
	bucket *s3.Bucket
	opened bool
}

// New returns a S3 based store.
func New(bucketID string, accessKey string, secretKey string) chainstore.Store {
	return &s3Store{BucketID: bucketID, AccessKey: accessKey, SecretKey: secretKey}
}

func (s *s3Store) Open() (err error) {
	if s.opened {
		return
	}

	auth, err := aws.GetAuth(s.AccessKey, s.SecretKey)
	if err != nil {
		return
	}

	s.conn = s3.New(auth, aws.USEast) // TODO: hardcoded region..?
	s.conn.HTTPClient = func() *http.Client {
		c := &http.Client{}
		return c
	}
	s.bucket = s.conn.Bucket(s.BucketID)
	s.opened = true
	return
}

func (s *s3Store) Close() (err error) {
	s.opened = false
	return // TODO: .. nothing to do here..?
}

func (s *s3Store) Put(ctx context.Context, key string, val []byte) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// TODO: configurable options for acl when making new s3 store
		return s.bucket.Put(key, val, `application/octet-stream`, s3.PublicRead)
	}
}

func (s *s3Store) Get(ctx context.Context, key string) (val []byte, err error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		val, err = s.bucket.Get(key)
		if err != nil {
			s3err, ok := err.(*s3.Error)
			if ok && s3err.Code != "NoSuchKey" {
				return nil, err
			}
		}
		return val, nil
	}
}

func (s *s3Store) Del(ctx context.Context, key string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return s.bucket.Del(key)
	}
}
