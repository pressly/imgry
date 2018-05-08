package server

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/pressly/imgry"
	"github.com/pressly/imgry/imagick"
	"github.com/pressly/lg"
)

var (
	EmptyImageKey = fmt.Sprintf("%x", sha1.Sum([]byte("")))

	ErrInvalidImageKey = errors.New("invalid image key")
)

// TODO: we should probably keep the Sizing as a url.Values and store it in the Hash value separately..
// then we can query for things... ie. find key, with whatever sizing props... etc..

type Image struct {
	Key         string        `json:"key" redis:"key"`
	SrcURL      string        `json:"src_url" redis:"src"`
	Width       int           `json:"width" redis:"w"`
	Height      int           `json:"height" redis:"h"`
	Format      string        `json:"format" redis:"f"`
	SizingQuery string        `json:"-" redis:"q"` // query from below, for saving
	Sizing      *imgry.Sizing `json:"-" redis:"-"`
	Data        []byte        `json:"-" redis:"-"`

	img imgry.Image
}

func (im *Image) genKey() {
	if im.SrcURL != "" {
		im.Key = sha1hash([]byte(im.SrcURL))
		return
	}

	if len(im.Data) > 0 {
		im.Key = sha1hash(im.Data)
		return
	}

	im.Key = EmptyImageKey
}

func sha1hash(in []byte) string {
	sum := sha1.Sum(in)
	return base64.RawURLEncoding.EncodeToString(sum[0:])
}

func brokenSha1hash(in string) string {
	hasher := sha1.New()
	fmt.Fprintf(hasher, in)
	return hex.EncodeToString(hasher.Sum(nil))
}

// Make sure to call Release() if methods LoadImage(), SizeIt()
// or MakeSize() are called.

func (im *Image) LoadImage() (err error) {
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
			lg.Fatalf("**** ENGINE FAILURE on %s", im.SrcURL)
		}
		return err
	}

	im.sync()

	return nil
}

func (im *Image) SrcFilename() string {
	fname := strings.Split(im.SrcURL, "?")
	fname = strings.Split(fname[0], "/")
	return fname[len(fname)-1]
}

func (im *Image) SrcFileExtension() string {
	parts := strings.Split(im.SrcFilename(), ".")
	ext := strings.ToLower(parts[len(parts)-1])
	return ext
}

func (im *Image) IsValidImage() bool {
	return im != nil &&
		im.Width > 0 &&
		im.Height > 0 &&
		im.Format != "" &&
		len(im.Data) > 0
}

// Sizes the current image in place
func (im *Image) SizeIt(sizing *imgry.Sizing) error {
	if err := im.ValidateKey(); err != nil {
		return err
	}

	if im.img == nil {
		if err := im.LoadImage(); err != nil {
			return err
		}
	}

	err := im.img.SizeIt(sizing)
	if err != nil {
		return fmt.Errorf("Error occurred when sizing an image: %s", err)
	}

	im.sync()

	return nil
}

// Create a new blob object from an existing size
func (im *Image) MakeSize(sizing *imgry.Sizing) (*Image, error) {
	if err := im.ValidateKey(); err != nil {
		return nil, err
	}

	im2 := &Image{
		Key:         im.Key,
		Data:        im.Data,
		SrcURL:      im.SrcURL,
		Sizing:      sizing,
		SizingQuery: sizing.ToQuery().Encode(),
	}

	// Clone the originating image
	var err error

	if err = im2.LoadImage(); err != nil {
		return nil, err
	}

	// Resize the new image object
	if err = im2.SizeIt(sizing); err != nil {
		return nil, err
	}

	return im2, nil
}

func (im *Image) ValidateKey() error {
	if im.Key == "" || im.Key == EmptyImageKey {
		return ErrInvalidImageKey
	}
	return nil
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

func (im *Image) Info() imgry.ImageInfo {
	if im == nil {
		return imgry.ImageInfo{}
	}
	im.sync()

	return imgry.ImageInfo{
		URL:           im.SrcURL,
		Format:        im.Format,
		Mimetype:      im.MimeType(),
		Width:         im.Width,
		Height:        im.Height,
		AspectRatio:   float64(im.Width) / float64(im.Height),
		ContentLength: len(im.Data),
	}
}
