package server

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"time"

	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"
	"github.com/rcrowley/go-metrics"
	"github.com/zenazn/goji/web"
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

type loggedResponseWriter struct {
	http.ResponseWriter
	status int
}

func (l *loggedResponseWriter) WriteHeader(status int) {
	l.status = status
	l.ResponseWriter.WriteHeader(status)
}

func (l *loggedResponseWriter) Status() int {
	return l.status
}

func trackRoute(metricID string) func(*web.C, http.Handler) http.Handler {
	return func(c *web.C, h http.Handler) http.Handler {
		route := fmt.Sprintf("route.%s", metricID)
		routeTimer := metrics.GetOrRegisterTimer(route, nil)
		errCounter := metrics.GetOrRegisterCounter(fmt.Sprintf("%s-err", route), nil)
		handler := func(w http.ResponseWriter, r *http.Request) {
			reqStart := time.Now()

			lw := &loggedResponseWriter{w, -1}
			h.ServeHTTP(lw, r)

			routeTimer.UpdateSince(reqStart)
			if lw.Status() >= 400 {
				errCounter.Inc(1)
			}
		}
		return http.HandlerFunc(handler)
	}
}
