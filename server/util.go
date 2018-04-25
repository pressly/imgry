package server

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/pressly/imgry"
	"github.com/unrolled/render"
)

func S3Client(s S3Bucket) *s3.S3 {
	cfg := &aws.Config{
		Region: aws.String(s.S3Region),
	}
	session := session.New(cfg)

	if s.S3AccessKey != "" {
		session.Config.WithCredentials(credentials.NewStaticCredentials(s.S3AccessKey, s.S3SecretKey, ""))
	} else {
		session.Config.WithCredentials(ec2rolecreds.NewCredentials(session))
	}

	return s3.New(session)
}

// Upload a file to S3 storage and return the url
func S3Upload(prefix string, im *Image) (string, error) {
	path := fmt.Sprintf("/%s/uploads/%s.%s", prefix, im.Key, im.Format)
	resp := ""
	for _, s := range app.Config.Chainstore.S3Buckets {
		params := &s3.PutObjectInput{
			Bucket:      aws.String(s.S3Bucket),
			Key:         aws.String(path),
			ACL:         aws.String(s3.ObjectCannedACLPublicRead),
			ContentType: aws.String(im.MimeType()),
			Body:        bytes.NewReader(im.Data),
		}

		c := S3Client(s)
		_, err := c.PutObject(params)
		if err != nil {
			return "", err
		}
		resp = fmt.Sprintf("%s/%s%s", c.ClientInfo.Endpoint, s.S3Bucket, path)
	}

	return resp, nil
}

type Responder struct {
	*render.Render
}

func NewResponder() *Responder {
	return &Responder{render.New(render.Options{})}
}

func (r *Responder) ImageError(w http.ResponseWriter, status int, err error) {
	if err == nil {
		r.Data(w, status, []byte{})
		return
	}

	r.cacheErrors(w, err)
	w.Header().Set("X-Err", err.Error())
	r.Data(w, status, []byte{})
}

func (r *Responder) ImageInfo(w http.ResponseWriter, status int, im *Image) {
	w.Header().Set("Content-Type", im.MimeType())
	w.Header().Set("X-Meta-Width", fmt.Sprintf("%d", im.Width))
	w.Header().Set("X-Meta-Height", fmt.Sprintf("%d", im.Height))
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", app.Config.CacheMaxAge))
	w.Header().Set("Last-Modified", time.Now().Format(http.TimeFormat))

	r.JSON(w, status, im)
}

func (r *Responder) Image(w http.ResponseWriter, status int, im *Image) {
	w.Header().Set("Content-Type", im.MimeType())
	w.Header().Set("X-Meta-Width", fmt.Sprintf("%d", im.Width))
	w.Header().Set("X-Meta-Height", fmt.Sprintf("%d", im.Height))
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", app.Config.CacheMaxAge))
	w.Header().Set("Last-Modified", time.Now().Format(http.TimeFormat))

	r.Data(w, status, im.Data)
}

func (r *Responder) ApiError(w http.ResponseWriter, status int, err error) {
	if err == nil {
		r.JSON(w, status, []byte{})
		return
	}

	r.cacheErrors(w, err)
	r.JSON(w, status, map[string]interface{}{"error": err.Error()})
}

func (r *Responder) cacheErrors(w http.ResponseWriter, err error) {
	switch err {
	case imgry.ErrInvalidImageData, ErrInvalidURL:
		// For invalid inputs, we tell the surrogate to cache the
		// error for a small amount of time.
		w.Header().Set("Cache-Control", "s-maxage=300") // 5 minutes
	default:
	}
}
