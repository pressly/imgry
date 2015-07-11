package s3store

import (
	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"
)

type s3Store struct {
	BucketId, AccessKey, SecretKey string

	conn   *s3.S3
	bucket *s3.Bucket
	opened bool
}

func New(bucketId string, accessKey string, secretKey string) *s3Store {
	return &s3Store{BucketId: bucketId, AccessKey: accessKey, SecretKey: secretKey}
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
	s.bucket = s.conn.Bucket(s.BucketId)
	s.opened = true
	return
}

func (s *s3Store) Close() (err error) {
	s.opened = false
	return // TODO: .. nothing to do here..?
}

func (s *s3Store) Put(key string, val []byte) error {
	// TODO: configurable options for acl when making new s3 store
	return s.bucket.Put(key, val, `application/octet-stream`, s3.PublicRead)
}

func (s *s3Store) Get(key string) (val []byte, err error) {
	val, err = s.bucket.Get(key)
	if err != nil {
		s3err, ok := err.(*s3.Error)
		if ok && s3err.Code != "NoSuchKey" {
			return nil, err
		}
	}
	return val, nil
}

func (s *s3Store) Del(key string) error {
	return s.bucket.Del(key)
}
