package imagex

import (
	"bytes"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"

	_ "github.com/jsummers/gobmp"
	// _ "github.com/nl5887/golang-image/jpeg"
	"github.com/pressly/imgry"
	// "github.com/nfnt/resize"
	// "github.com/oliamb/cutter"
	"github.com/pressly/goico"
)

/* TODO:
1. Resize flat image
2. Resize animated gif... frames..?
*/

var (
	ErrEngineReleased = errors.New("imgry: engine has been released.")
)

func init() {
	// Manually register the ICO format as it has no magic string and 0010 is a
	// terrible thing to sniff for. This init is run after the imports' inits in
	// this package. So GIF, JPEG, PNG, BMP will be init and registered first.
	// This means always leave the _ imports for images above.
	image.RegisterFormat("ico", "\x00\x00\x01\x00", ico.Decode, ico.DecodeConfig)
}

type Engine struct{}

func (ng Engine) LoadFile(filename string) (imgry.Image, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return ng.LoadBlob(b)
}

func (ng Engine) LoadBlob(b []byte) (imgry.Image, error) {
	img, format, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	im := &Image{}
	im.data = b
	im.width = uint(img.Bounds().Dx())
	im.height = uint(img.Bounds().Dy())
	im.format = format

	return im, nil
}

type Image struct {
	// mw *imagick.MagickWand

	data   []byte
	width  uint
	height uint
	format string
}

func (i *Image) Data() []byte {
	return i.data
}

func (i *Image) Width() uint {
	return i.width
}

func (i *Image) Height() uint {
	return i.height
}

func (i *Image) Format() string {
	return i.format
}

func (i *Image) SetFormat(format string) error {
	if i.Released() {
		return ErrEngineReleased
	}
	// if err := i.mw.SetImageFormat(format); err != nil {
	// 	return err
	// }
	if err := i.sync(); err != nil {
		return err
	}
	return nil
}

func (i *Image) Released() bool {
	// return i.mw == nil
	return true
}

func (i *Image) Release() {
	// if i.mw != nil {
	//   i.mw.Destroy()
	//   i.mw = nil
	// }
}

func (i *Image) Clone() imgry.Image {
	// TODO
	return nil
}

func (i *Image) SizeIt(sizing *imgry.Sizing) error {
	if i.Released() {
		return ErrEngineReleased
	}

	return nil
}

func (i *Image) WriteToFile(fn string) error {
	err := ioutil.WriteFile(fn, i.Data(), 0664)
	return err
}

func (i *Image) sync() error {
	return nil
}
