package s3store

import (
	"bytes"
	"context"
	"io/ioutil"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pressly/chainstore"
)

type Config struct {
	S3Bucket    string
	S3AccessKey string
	S3SecretKey string
	S3Region    string
	KMSKeyID    string
}

type s3Store struct {
	conf Config

	conn   *s3.S3
	bucket *s3.Bucket
	opened bool
}

// New returns a S3 based store.
func New(conf Config) chainstore.Store {
	return &s3Store{conf: conf}
}

func (s *s3Store) Open() error {
	cfg := &aws.Config{
		Region: &s.conf.S3Region,
	}
	session := session.New(cfg)

	if s.conf.S3AccessKey != "" {
		session.Config.WithCredentials(credentials.NewStaticCredentials(s.conf.S3AccessKey, s.conf.S3SecretKey, ""))
	} else {
		session.Config.WithCredentials(ec2rolecreds.NewCredentials(session))
	}

	s.conn = s3.New(session)

	return nil
}

func (s *s3Store) Close() (err error) {
	return
}

func (s *s3Store) Put(ctx context.Context, key string, val []byte) error {
	params := &s3.PutObjectInput{
		Bucket:      aws.String(s.conf.S3Bucket),
		Key:         aws.String(key),
		ACL:         aws.String("private"),
		ContentType: aws.String(`application/octet-stream`),
		Body:        bytes.NewReader(val),
	}

	if s.conf.KMSKeyID != "" {
		params.SetSSEKMSKeyId(s.conf.KMSKeyID)
		params.SetServerSideEncryption(s3.ServerSideEncryptionAwsKms)
	}

	_, err := s.conn.PutObjectWithContext(aws.Context(context.Background()), params)
	return err
}

func (s *s3Store) Get(ctx context.Context, key string) ([]byte, error) {
	params := &s3.GetObjectInput{
		Bucket: aws.String(s.conf.S3Bucket),
		Key:    aws.String(key),
	}

	resp, err := s.conn.GetObjectWithContext(aws.Context(ctx), params)
	if err != nil {
		aerr, ok := err.(awserr.Error)
		if ok && aerr.Code() == s3.ErrCodeNoSuchKey {
			return nil, nil
		}
		return nil, err
	}

	val, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return val, nil
}

func (s *s3Store) Del(ctx context.Context, key string) error {
	params := s3.DeleteObjectInput{
		Bucket: aws.String(s.conf.S3Bucket),
		Key:    aws.String(key),
	}

	_, err := s.conn.DeleteObjectWithContext(aws.Context(ctx), &params)
	aerr, ok := err.(awserr.Error)
	if ok && aerr.Code() == s3.ErrCodeNoSuchKey {
		return nil
	}
	return err
}
