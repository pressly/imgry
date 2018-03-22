package server

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"time"

	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"
	"github.com/pressly/imgry"
	"github.com/unrolled/render"
)

func getS3Bucket(accessKey, secretKey, bucket string) *s3.Bucket {
	auth := aws.Auth{
		AccessKey: accessKey,
		SecretKey: secretKey,
	}
	return s3.New(auth, aws.USEast).Bucket(bucket)
}

func s3Path(path string, data []byte, ext string) string {
	h := md5.Sum(data)
	return fmt.Sprintf("/%s/uploads/%x.%s", path, h, ext)
}

func s3Upload(bucket *s3.Bucket, path string, im *Image) (string, error) {
	var url string
	if len(im.Data) == 0 {
		return "", fmt.Errorf("No image data found for %s", path)
	}
	err := bucket.Put(path, im.Data, im.MimeType(), s3.PublicRead)
	if err != nil {
		return url, err
	}
	url = bucket.URL(path)
	return url, nil
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
