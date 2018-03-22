package server

import (
	"context"
	"net/http"

	"github.com/goware/urlx"
	"github.com/pressly/chi"
)

type contextKey struct {
	name string
}

type wrappedResponseWriter struct {
	http.ResponseWriter
	status int
}

var (
	bucketCtxKey = contextKey{"bucket"}
	imageCtxKey  = contextKey{"imageURL"}
	sizingCtxKey = contextKey{"imageSizing"}
)

func BucketURLCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if b := chi.URLParam(r, "bucket"); b != "" {
			bucket, err := NewBucket(b)
			if err != nil {
				respond.ImageError(w, 422, err)
				return
			}
			ctx = context.WithValue(ctx, bucketCtxKey, bucket)
		}

		if i, ok := r.URL.Query()["url"]; ok {
			u, err := urlx.Parse(i[0])
			if err != nil {
				respond.ImageError(w, 422, ErrInvalidURL)
				return
			}

			ctx = context.WithValue(ctx, imageCtxKey, u.String())
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
