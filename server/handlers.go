package server

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/pressly/imgry"
	"github.com/pressly/imgry/imagick"
	"github.com/pressly/lg"
)

var (
	MimeTypes = map[string]string{
		"png":  "image/png",
		"jpeg": "image/jpeg",
		"jpg":  "image/jpeg",
		"bmp":  "image/bmp",
		"bm":   "image/bmp",
		"gif":  "image/gif",
		"ico":  "image/x-icon",
	}

	ErrInvalidURL = errors.New("invalid url")
)

func BucketGetIndex(w http.ResponseWriter, r *http.Request) {
	if r.Context().Value(imageCtxKey) == nil {
		Index(w, r)
		return
	}
	BucketFetchItem(w, r)
}

func BucketFetchItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	bucket, _ := ctx.Value(bucketCtxKey).(*Bucket)
	fetchURL, _ := ctx.Value(imageCtxKey).(string)

	// First check if we have the original.. a bit of extra overhead, but its okay
	_, err := bucket.DbFindImage(ctx, fetchURL, nil)
	if err != nil && err != ErrImageNotFound {
		respond.ImageError(w, 422, err)
		return
	}

	// Fetch the image on-demand and add to bucket if we dont have it
	if err == ErrImageNotFound {
		// TODO: add image sizing throttler here....

		_, err := bucket.AddImagesFromUrls(ctx, []string{fetchURL})
		if err != nil {
			lg.Errorf("Fetching failed for %s because %s", fetchURL, err)
			respond.ImageError(w, 422, err)
			return
		}
	}

	BucketGetItem(w, r)
}

func BucketGetItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	bucket, _ := ctx.Value(bucketCtxKey).(*Bucket)
	fetchURL, _ := ctx.Value(imageCtxKey).(string)

	sizing, err := imgry.NewSizingFromQuery(r.URL.RawQuery)
	if err != nil {
		lg.Errorf("Failed to create sizing for %s cause: %s", r.URL, err)
		respond.ImageError(w, 422, err)
		return
	}

	im, err := bucket.GetImageSize(ctx, fetchURL, sizing)
	if err != nil {
		lg.Errorf("Failed to get image for %s cause: %s", r.URL, err)
		respond.ImageError(w, 422, err)
		return
	}

	// If requested, only return the image details instead of the data
	if r.URL.Query().Get("info") != "" {
		respond.ImageInfo(w, 200, im)
		return
	}

	respond.Image(w, 200, im)
}

// TODO: this can be optimized significantly..........
// Ping / DecodeConfig ... do we have to use image magick.......?
func GetImageInfo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	fetchURL, _ := ctx.Value(imageCtxKey).(string)

	response, err := app.Fetcher.Get(ctx, fetchURL)
	if err != nil {
		respond.ApiError(w, 422, err)
		return
	}

	ng := imagick.Engine{}
	info, err := ng.GetImageInfo(response.Data)
	if err != nil {
		respond.ApiError(w, 422, err)
		return
	}
	info.URL = response.URL.String()
	info.Mimetype = MimeTypes[info.Format]

	w.Header().Set("X-Meta-Width", fmt.Sprintf("%d", info.Width))
	w.Header().Set("X-Meta-Height", fmt.Sprintf("%d", info.Height))
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", app.Config.CacheMaxAge))
	respond.JSON(w, 200, info)
}

// Image upload to an s3 bucket, respond with a direct url to the uploaded
// image. Avoid using respond.ApiError() here to prevent any of the responses
// from being cached.
func BucketImageUpload(w http.ResponseWriter, r *http.Request) {
	var err error
	var data []byte

	im := &Image{Data: data}
	defer im.Release()

	ctx := r.Context()
	bucket, _ := ctx.Value(bucketCtxKey).(*Bucket)

	file, _, err := r.FormFile("file")
	switch err {
	case nil:
		defer file.Close()
		im.Data, err = ioutil.ReadAll(file)
		if err != nil {
			respond.JSON(w, 422, map[string]interface{}{"error": err.Error()})
			return
		}

	case http.ErrMissingFile:
		base64file := r.FormValue("base64file")
		fileLen := len(base64file)
		if fileLen < 100 {
			respond.JSON(w, 422, map[string]interface{}{"error": "invalid file upload"})
			return
		}
		data, err = base64.StdEncoding.DecodeString(base64file)
		if err != nil {
			respond.JSON(w, 422, map[string]interface{}{"error": err.Error()})
			return
		}

	default:
		respond.JSON(w, 422, map[string]interface{}{"error": err.Error()})
		return
	}

	if err = im.LoadImage(); err != nil {
		respond.JSON(w, 422, map[string]interface{}{"error": err.Error()})
		return
	}

	if err := bucket.UploadImage(ctx, im); err != nil {
		respond.JSON(w, 422, map[string]interface{}{"error": err.Error()})
		return
	}

	respond.JSON(w, 200, im.Info())
}

func BucketAddItems(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	bucket, _ := ctx.Value(bucketCtxKey).(*Bucket)

	urls := r.URL.Query()["url[]"]
	if fetchURL, ok := ctx.Value(imageCtxKey).(string); ok {
		urls = append(urls, fetchURL)
	}

	if len(urls) == 0 {
		respond.JSON(w, 422, map[string]interface{}{"error": "Url or urls parameter required"})
		return
	}

	images, err := bucket.AddImagesFromUrls(ctx, urls)
	if err != nil {
		// TODO: refactor.. ApiError will cache invalid image errors,
		// but for an array of urls, we shouldn't cache the entire response
		if len(urls) == 1 {
			respond.ApiError(w, 422, err)
		} else {
			respond.JSON(w, 422, map[string]interface{}{"error": err.Error()})
		}
		return
	}

	respond.JSON(w, 200, images)
}

func BucketDeleteItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	bucket, _ := ctx.Value(bucketCtxKey).(*Bucket)
	fetchURL, _ := ctx.Value(imageCtxKey).(string)

	if err := bucket.DbDelImage(ctx, fetchURL); err != nil {
		respond.JSON(w, 422, map[string]interface{}{"error": err.Error()})
		return
	}

	respond.JSON(w, 200, []byte{})
}
