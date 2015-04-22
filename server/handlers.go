package server

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/pressly/imgry/imagick"

	"github.com/goware/urlx"
	"github.com/pressly/imgry"
	"github.com/zenazn/goji/web"
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

func BucketGetIndex(c web.C, w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("url") == "" {
		respond.Data(w, 200, []byte{})
		return
	}
	BucketFetchItem(c, w, r)
}

func BucketGetItem(c web.C, w http.ResponseWriter, r *http.Request) {
	// TODO: bucket binding should happen in a middleware... refactor all handlers
	// that use it..
	bucket, err := NewBucket(c.URLParams["bucket"])
	if err != nil {
		lg.Error("Failed to create bucket for %s cause: %s", r.URL, err)
		respond.ImageError(w, 422, err)
		return
	}

	sizing, err := imgry.NewSizingFromQuery(r.URL.RawQuery)
	if err != nil {
		lg.Error("Failed to create sizing for %s cause: %s", r.URL, err)
		respond.ImageError(w, 422, err)
		return
	}

	im, err := bucket.GetImageSize(c.URLParams["key"], sizing)
	if err != nil {
		lg.Error("Failed to get image for %s cause: %s", r.URL, err)
		respond.ImageError(w, 422, err)
		return
	}

	w.Header().Set("Content-Type", im.MimeType())
	w.Header().Set("X-Meta-Width", fmt.Sprintf("%d", im.Width))
	w.Header().Set("X-Meta-Height", fmt.Sprintf("%d", im.Height))
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", app.Config.Server.CacheMaxAge))

	// If requested, only return the image details instead of the data
	if r.URL.Query().Get("info") != "" {
		// TODO: eventually, once the ruby stack is updated, we should
		// return an ImageInfo packet here instead..
		respond.JSON(w, http.StatusOK, im)
		return
	}

	respond.Data(w, 200, im.Data)
}

func BucketFetchItem(c web.C, w http.ResponseWriter, r *http.Request) {
	bucket, err := NewBucket(c.URLParams["bucket"])
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
	c.URLParams["key"] = imKey

	// First check if we have the original.. a bit of extra overhead, but its okay
	_, err = bucket.DbFindImage(imKey, nil)
	if err != nil && err != ErrImageNotFound {
		respond.ImageError(w, 422, err)
		return
	}

	// lg.Info("BucketFetchItem, after DbFindImage... %v", err)

	// Fetch the image on-demand and add to bucket if we dont have it
	if err == ErrImageNotFound {
		_, err := bucket.AddImagesFromUrls([]string{fetchUrl})
		if err != nil {
			lg.Error("Fetching failed for %s because %s", fetchUrl, err)
			respond.ImageError(w, 422, err)
			return
		}
	}

	BucketGetItem(c, w, r)
}

// TODO: this can be optimized significantly..........
// Ping / DecodeConfig ... do we have to use image magick.......?
func GetImageInfo(c web.C, w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		respond.ApiError(w, 422, errors.New("no image url"))
		return
	}

	response, err := app.HttpFetcher.Get(url)
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
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", app.Config.Server.CacheMaxAge))
	respond.JSON(w, 200, imfo)
}

// Image upload to an s3 bucket, respond with a direct url to the uploaded
// image. Avoid using respond.ApiError() here to prevent any of the responses
// from being cached.
func BucketImageUpload(c web.C, w http.ResponseWriter, r *http.Request) {
	var url string
	var err error

	file, header, err := r.FormFile("file")
	if err != nil {
		respond.JSON(w, 422, map[string]interface{}{"error": err.Error()})
		return
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		respond.JSON(w, 422, map[string]interface{}{"error": err.Error()})
		return
	}

	im := NewImageFromSrcUrl(header.Filename)
	defer im.Release()

	im.Data = data
	if err = im.LoadImage(); err != nil {
		respond.JSON(w, 422, map[string]interface{}{"error": err.Error()})
		return
	}

	s3Bucket := getS3Bucket(app.Config.Chainstore.S3AccessKey,
		app.Config.Chainstore.S3SecretKey,
		app.Config.Chainstore.S3Bucket)

	path := s3Path(c.URLParams["bucket"], im.Data, im.Format)

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

func BucketAddItems(c web.C, w http.ResponseWriter, r *http.Request) {
	bucket, err := NewBucket(c.URLParams["bucket"])
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

	images, err := bucket.AddImagesFromUrls(urls)
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

func BucketDeleteItem(c web.C, w http.ResponseWriter, r *http.Request) {
	bucket, err := NewBucket(c.URLParams["bucket"])
	if err != nil {
		respond.JSON(w, 422, map[string]interface{}{"error": err.Error()})
		return
	}

	pUrl := r.URL.Query().Get("url")
	if pUrl != "" {
		pKey := sha1Hash(pUrl) // transform to what is expected..
		c.URLParams["key"] = pKey
	}
	imageKey := c.URLParams["key"]
	if imageKey == "" {
		respond.JSON(w, 422, map[string]interface{}{
			"error": "Unable to determine the key for the delete operation",
		})
		return
	}

	err = bucket.DbDelImage(imageKey)
	if err != nil {
		respond.JSON(w, 422, map[string]interface{}{"error": err.Error()})
		return
	}

	respond.JSON(w, 200, []byte{})
}

// DEPRECATED
func BucketV0FetchItem(c web.C, w http.ResponseWriter, r *http.Request) {
	if c.URLParams == nil {
		c.URLParams = make(map[string]string)
	}
	c.URLParams["bucket"] = "tmp" // we imply the bucket name..
	BucketFetchItem(c, w, r)
}
