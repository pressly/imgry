package server

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/pressly/imgry"
	"github.com/pressly/imgry/imagick"
	"github.com/rcrowley/go-metrics"
)

// TODO: we should probably keep the Sizing as a url.Values and store it in the Hash value separately..
// then we can query for things... ie. find key, with whatever sizing props... etc..

type Image struct {
	Key         string        `json:"key" redis:"key"`
	SrcUrl      string        `json:"src_url" redis:"src"`
	Width       int           `json:"width" redis:"w"`
	Height      int           `json:"height" redis:"h"`
	Format      string        `json:"format" redis:"f"`
	SizingQuery string        `json:"-" redis:"q"` // query from below, for saving
	Sizing      *imgry.Sizing `json:"-" redis:"-"`
	Data        []byte        `json:"-" redis:"-"`

	img imgry.Image
}

// Hrmm.. how will we generate a Uid if we just have a blob and no srcurl..?
// perhaps we allow the uid to be like "something.jpg" if they want..?
// unlikely to be collisions anyways...
// how to dedupe those..? guess we cant.. only if it was based on blob..
func NewImageFromKey(key string) *Image {
	return &Image{Key: key}
}

func NewImageFromSrcUrl(srcUrl string) *Image {
	return &Image{SrcUrl: srcUrl, Key: sha1Hash(srcUrl)}
}

func sha1Hash(in string) string {
	hasher := sha1.New()
	fmt.Fprintf(hasher, in)
	return hex.EncodeToString(hasher.Sum(nil))
}

// Make sure to call Release() if methods LoadImage(), SizeIt()
// or MakeSize() are called.

func (im *Image) LoadImage() (err error) {
	m := metrics.GetOrRegisterTimer("fn.image.LoadBlob", nil) // TODO: update metric name
	defer m.UpdateSince(time.Now())

	// TODO: throttle the number of images we load at a given time..
	// this should be configurable...

	defer func() {
		if r := recover(); r != nil {
			var ok bool
			err, ok = r.(error)
			if !ok {
				err = fmt.Errorf("imgry: %v", r)
			}
		}
	}()

	var formatHint string
	if im.SrcFileExtension() == "ico" {
		formatHint = "ico"
	}

	ng := imagick.Engine{}
	im.img, err = ng.LoadBlob(im.Data, formatHint)
	if err != nil {
		if err == imagick.ErrEngineFailure {
			lg.Error("**** ENGINE FAILURE on %s", im.SrcUrl)
		}
		return err
	}

	im.sync()

	return nil
}

func (im *Image) SrcFilename() string {
	fname := strings.Split(im.SrcUrl, "?")
	fname = strings.Split(fname[0], "/")
	return fname[len(fname)-1]
}

func (im *Image) SrcFileExtension() string {
	parts := strings.Split(im.SrcFilename(), ".")
	ext := strings.ToLower(parts[len(parts)-1])
	return ext
}

func (im *Image) IsValidImage() bool {
	return im.Width > 0 && im.Height > 0 && im.Format != ""
}

// Sizes the current image in place
func (im *Image) SizeIt(sizing *imgry.Sizing) error {
	m := metrics.GetOrRegisterTimer("fn.image.SizeIt", nil)
	defer m.UpdateSince(time.Now())

	var err error

	if im.img == nil {
		if err := im.LoadImage(); err != nil {
			return err
		}
	}

	err = im.img.SizeIt(sizing)
	if err != nil {
		return fmt.Errorf("Error occurred when sizing an image: %s", err)
	}

	im.sync()

	return nil
}

// Create a new blob object from an existing size
func (im *Image) MakeSize(sizing *imgry.Sizing) (*Image, error) {
	m := metrics.GetOrRegisterTimer("fn.image.MakeSize", nil)
	defer m.UpdateSince(time.Now())

	var err error

	im2 := &Image{
		Key:         im.Key,
		Data:        im.Data,
		SrcUrl:      im.SrcUrl,
		Sizing:      sizing,
		SizingQuery: sizing.ToQuery().Encode(),
	}

	// Clone the originating image
	if err = im2.LoadImage(); err != nil {
		return nil, err
	}

	// Resize the new image object
	if err = im2.SizeIt(sizing); err != nil {
		return nil, err
	}

	return im2, nil
}

func (im *Image) MimeType() string {
	mt := MimeTypes[im.Format]
	if mt == "" {
		mt = "application/octet-stream"
	}
	return mt
}

func (im *Image) Release() {
	if im == nil {
		return
	}
	m := metrics.GetOrRegisterCounter("fn.image.Release", nil)
	defer m.Inc(1)

	if im.img != nil {
		im.img.Release()
	}
}

func (im *Image) sync() {
	im.Width = im.img.Width()
	im.Height = im.img.Height()
	im.Format = im.img.Format()
	im.Data = im.img.Data()
}
