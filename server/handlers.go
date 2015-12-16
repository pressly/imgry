package server

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/goware/lg"
	"github.com/goware/urlx"
	"github.com/pressly/chi"
	"github.com/pressly/imgry"
	"github.com/pressly/imgry/imagick"
	"golang.org/x/net/context"
)

var (
	MimeTypes = map[string]string{
		"mp4":  "video/mp4",
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

func BucketGetIndex(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("url") == "" {
		respond.Data(w, 200, []byte{})
		return
	}
	BucketFetchItem(ctx, w, r)
}

func BucketFetchItem(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	bucket, err := NewBucket(chi.URLParams(ctx)["bucket"])
	if err != nil {
		respond.ImageError(w, 422, err)
		return
	}

	fetchUrl := r.URL.Query().Get("url")
	if fetchUrl == "" {
		respond.ImageError(w, 422, ErrInvalidURL)
		return
	}

	u, err := urlx.Parse(fetchUrl)
	if err != nil {
		respond.ImageError(w, 422, ErrInvalidURL)
		return
	}
	fetchUrl = u.String()

	imKey := sha1Hash(fetchUrl) // transform to what is expected..
	chi.URLParams(ctx)["key"] = imKey

	// First check if we have the original.. a bit of extra overhead, but its okay
	_, err = bucket.DbFindImage(ctx, imKey, nil)
	if err != nil && err != ErrImageNotFound {
		respond.ImageError(w, 422, err)
		return
	}

	// Fetch the image on-demand and add to bucket if we dont have it
	if err == ErrImageNotFound {
		// TODO: add image sizing throttler here....

		_, err := bucket.AddImagesFromUrls(ctx, []string{fetchUrl})
		if err != nil {
			lg.Errorf("Fetching failed for %s because %s", fetchUrl, err)
			respond.ImageError(w, 422, err)
			return
		}
	}

	BucketGetItem(ctx, w, r)
}

func BucketGetItem(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// TODO: bucket binding should happen in a middleware... refactor all handlers
	// that use it..
	bucket, err := NewBucket(chi.URLParams(ctx)["bucket"])
	if err != nil {
		lg.Errorf("Failed to create bucket for %s cause: %s", r.URL, err)
		respond.ImageError(w, 422, err)
		return
	}

	sizing, err := imgry.NewSizingFromQuery(r.URL.RawQuery)
	if err != nil {
		lg.Errorf("Failed to create sizing for %s cause: %s", r.URL, err)
		respond.ImageError(w, 422, err)
		return
	}

	im, err := bucket.GetImageSize(ctx, chi.URLParams(ctx)["key"], sizing)
	if err != nil {
		lg.Errorf("Failed to get image for %s cause: %s", r.URL, err)
		respond.ImageError(w, 422, err)
		return
	}

	w.Header().Set("Content-Type", im.MimeType())
	w.Header().Set("X-Meta-Width", fmt.Sprintf("%d", im.Width))
	w.Header().Set("X-Meta-Height", fmt.Sprintf("%d", im.Height))
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", app.Config.CacheMaxAge))

	// If requested, only return the image details instead of the data
	if r.URL.Query().Get("info") != "" {
		// TODO: eventually, once the ruby stack is updated, we should
		// return an ImageInfo packet here instead..
		respond.JSON(w, http.StatusOK, im)
		return
	}

	respond.Data(w, 200, im.Data)
}

// TODO: this can be optimized significantly..........
// Ping / DecodeConfig ... do we have to use image magick.......?
func GetImageInfo(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		respond.ApiError(w, 422, errors.New("no image url"))
		return
	}

	response, err := app.Fetcher.Get(ctx, url)
	if err != nil {
		respond.ApiError(w, 422, err)
		return
	}
	data := response.Data

	ng := imagick.Engine{}
	imfo, err := ng.GetImageInfo(data)
	if err != nil {
		respond.ApiError(w, 422, err)
		return
	}
	imfo.URL = response.URL.String()
	imfo.Mimetype = MimeTypes[imfo.Format]

	w.Header().Set("X-Meta-Width", fmt.Sprintf("%d", imfo.Width))
	w.Header().Set("X-Meta-Height", fmt.Sprintf("%d", imfo.Height))
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", app.Config.CacheMaxAge))
	respond.JSON(w, 200, imfo)
}

// Image upload to an s3 bucket, respond with a direct url to the uploaded
// image. Avoid using respond.ApiError() here to prevent any of the responses
// from being cached.
func BucketImageUpload(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var url string
	var err error
	var data []byte
	var im *Image

	file, header, err := r.FormFile("file")
	switch err {
	case nil:
		defer file.Close()

		data, err = ioutil.ReadAll(file)
		if err != nil {
			respond.JSON(w, 422, map[string]interface{}{"error": err.Error()})
			return
		}
		im = NewImageFromSrcUrl(header.Filename)

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

		// balance collision chance vs hash time
		if fileLen > 10000 {
			fileLen = 10000
		}
		im = NewImageFromSrcUrl(string(base64file[0:fileLen]))

	default:
		respond.JSON(w, 422, map[string]interface{}{"error": err.Error()})
		return
	}

	defer im.Release()

	im.Data = data
	if err = im.LoadImage(); err != nil {
		respond.JSON(w, 422, map[string]interface{}{"error": err.Error()})
		return
	}

	s3Bucket := getS3Bucket(app.Config.Chainstore.S3AccessKey,
		app.Config.Chainstore.S3SecretKey,
		app.Config.Chainstore.S3Bucket)

	path := s3Path(chi.URLParams(ctx)["bucket"], im.Data, im.Format)

	url, err = s3Upload(s3Bucket, path, im)
	if err != nil {
		respond.JSON(w, 422, map[string]interface{}{"error": err.Error()})
		return
	}

	imfo := imgry.ImageInfo{
		URL:           url,
		Format:        im.Format,
		Mimetype:      im.MimeType(),
		Width:         im.Width,
		Height:        im.Height,
		AspectRatio:   float64(im.Width) / float64(im.Height),
		ContentLength: len(im.Data),
	}

	respond.JSON(w, 200, imfo)
}

func BucketAddItems(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	bucket, err := NewBucket(chi.URLParams(ctx)["bucket"])
	if err != nil {
		respond.JSON(w, 422, map[string]interface{}{"error": err.Error()})
		return
	}

	urls := r.URL.Query()["url[]"]
	urls = append(urls, r.URL.Query()["url"]...)

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

func BucketDeleteItem(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	bucket, err := NewBucket(chi.URLParams(ctx)["bucket"])
	if err != nil {
		respond.JSON(w, 422, map[string]interface{}{"error": err.Error()})
		return
	}

	pUrl := r.URL.Query().Get("url")
	if pUrl != "" {
		pKey := sha1Hash(pUrl) // transform to what is expected..
		chi.URLParams(ctx)["key"] = pKey
	}
	imageKey := chi.URLParams(ctx)["key"]
	if imageKey == "" {
		respond.JSON(w, 422, map[string]interface{}{
			"error": "Unable to determine the key for the delete operation",
		})
		return
	}

	err = bucket.DbDelImage(ctx, imageKey)
	if err != nil {
		respond.JSON(w, 422, map[string]interface{}{"error": err.Error()})
		return
	}

	respond.JSON(w, 200, []byte{})
}

// DEPRECATED
func BucketV0FetchItem(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	chi.URLParams(ctx)["bucket"] = "tmp" // we imply the bucket name..
	BucketFetchItem(ctx, w, r)
}
