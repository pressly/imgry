package server

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	"github.com/goware/go-metrics"
	"github.com/goware/lg"
	"github.com/pressly/imgry"
	"github.com/pressly/imgry/ffmpeg"
	"github.com/pressly/imgry/imagick"
)

var (
	EmptyImageKey = sha1Hash("")

	ErrInvalidImageKey = errors.New("invalid image key")
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

	img              imgry.Image
	conversionFormat string
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
	defer metrics.MeasureSince([]string{"fn.image.LoadImage"}, time.Now())

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
			lg.Fatalf("**** ENGINE FAILURE on %s", im.SrcUrl)
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
	defer metrics.MeasureSince([]string{"fn.image.SizeIt"}, time.Now())

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

	im.conversionFormat = sizing.Format

	if err = im.sync(); err != nil {
		return fmt.Errorf("Error occurred when syncing the image: %s", err)
	}

	return nil
}

// Create a new blob object from an existing size
func (im *Image) MakeSize(sizing *imgry.Sizing) (*Image, error) {
	defer metrics.MeasureSince([]string{"fn.image.MakeSize"}, time.Now())

	if err := im.ValidateKey(); err != nil {
		return nil, err
	}

	im2 := &Image{
		Key:         im.Key,
		Data:        im.Data,
		SrcUrl:      im.SrcUrl,
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
	defer metrics.IncrCounter([]string{"fn.image.Release"}, 1)

	if im.img != nil {
		im.img.Release()
	}
}

func (im *Image) sync() error {
	im.Width = im.img.Width()
	im.Height = im.img.Height()

	if im.Format == "mp4" && len(im.Data) != 0 {
		// No need to convert again.
		return nil
	}

	im.Format = im.img.Format()
	im.Data = im.img.Data()

	if im.conversionFormat == "mp4" && im.Format == "gif" {
		// I still can't find how to pipe images to ffmpeg's stdin, so for now we
		// need to output them to disk.
		outputGIF := tmpFile(im.Key + ".gif") // Better if mounted on tmpfs.
		outputMP4 := tmpFile(im.Key + ".mp4")

		if err := ioutil.WriteFile(outputGIF, im.img.Data(), 0664); err != nil {
			return err
		}
		defer os.RemoveAll(path.Dir(outputGIF))

		if err := ffmpeg.Convert(outputGIF, outputMP4); err != nil {
			return err
		}
		defer os.RemoveAll(path.Dir(outputMP4))

		buf, err := ioutil.ReadFile(outputMP4)
		if err != nil {
			return err
		}

		// I'm going to take over the data buffer.
		im.Data = buf
		im.Format = "mp4"
	}

	return nil
}

func tmpFile(name string) string {
	for {
		dirname, _ := ioutil.TempDir("", "tmp-")
		file := dirname + "/" + name
		_, err := os.Stat(file)
		if !os.IsExist(err) {
			return file
		}
	}
	panic("reached")
}
