package imagick

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/gographics/imagick/imagick"
	"github.com/pressly/imgry"
)

var (
	ErrEngineReleased = errors.New("imagick: engine has been released.")
	ErrEngineFailure  = errors.New("imagick: unable to request a MagickWand")
)

type Engine struct {
	tmpDir string
	// TODO: perhaps we have counter of wands here..
	// it ensures we'll always release them..
	// AvailableWands .. .. set to MaxSizers ..
	// then, NewMagickWand() will limit, or error..
	// perhaps we make a channel of these things..
	// then block.. + ctx will have a timeout if it wants to stop waiting..
	// we can log these timeouts etc..
}

func (ng Engine) Version() string {
	v, _ := imagick.GetVersion()
	return fmt.Sprintf("%s", v)
}

func (ng Engine) Initialize(tmpDir string) error {
	if tmpDir != "" {
		if err := os.MkdirAll(tmpDir, 0755); err != nil {
			return err
		}
		ng.tmpDir = tmpDir
		os.Setenv("MAGICK_TMPDIR", tmpDir)
		ng.SweepTmpDir()
	}
	imagick.Initialize()
	return nil
}

func (ng Engine) Terminate() {
	imagick.Terminate()
	ng.SweepTmpDir()
}

func (ng Engine) SweepTmpDir() error {
	if ng.tmpDir == "" {
		return nil
	}
	err := filepath.Walk(
		ng.tmpDir,
		func(path string, info os.FileInfo, err error) error {
			if ng.tmpDir == path {
				return nil // skip the root
			}
			if strings.Index(filepath.Base(path), "magick") >= 0 {
				if err = os.Remove(path); err != nil {
					return fmt.Errorf("failed to sweet engine tmpdir %s, because: %s", path, err)
				}
			}
			return nil
		},
	)
	if err != nil {
		return err
	}
	return nil
}

func (ng Engine) LoadFile(filename string, srcFormat ...string) (imgry.Image, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return ng.LoadBlob(b, srcFormat...)
}

func (ng Engine) LoadBlob(b []byte, srcFormat ...string) (imgry.Image, error) {
	if len(b) == 0 {
		return nil, imgry.ErrInvalidImageData
	}

	mw := imagick.NewMagickWand()
	if !mw.IsVerified() {
		return nil, ErrEngineFailure
	}

	// Offer a hint to the decoder of the file format
	if len(srcFormat) > 0 {
		f := srcFormat[0]
		if f != "" {
			mw.SetFormat(f)
		}
	}

	err := mw.ReadImageBlob(b)
	if err != nil {
		mw.Destroy()
		return nil, imgry.ErrInvalidImageData
	}

	// TODO: perhaps we pass the engine instance like Image{engine: i}
	im := &Image{mw: mw, data: b}
	if err := im.sync(); err != nil {
		mw.Destroy()
		return nil, err
	}

	return im, nil
}

func (ng Engine) GetImageInfo(b []byte, srcFormat ...string) (*imgry.ImageInfo, error) {
	if len(b) == 0 {
		return nil, imgry.ErrInvalidImageData
	}

	mw := imagick.NewMagickWand()
	defer mw.Destroy()

	if !mw.IsVerified() {
		return nil, ErrEngineFailure
	}

	err := mw.PingImageBlob(b)
	if err != nil {
		return nil, imgry.ErrInvalidImageData
	}

	w, h := int(mw.GetImageWidth()), int(mw.GetImageHeight())
	ar := float64(int(float64(w)/float64(h)*10000)) / 10000

	format := strings.ToLower(mw.GetImageFormat())
	if format == "jpeg" {
		format = "jpg"
	}

	imfo := &imgry.ImageInfo{
		Format: format, Width: w, Height: h,
		AspectRatio: ar, ContentLength: len(b),
	}

	return imfo, nil
}

type Image struct {
	mw *imagick.MagickWand

	data   []byte
	width  int
	height int
	format string
}

func (i *Image) Data() []byte {
	return i.data
}

func (i *Image) Width() int {
	return i.width
}

func (i *Image) Height() int {
	return i.height
}

func (i *Image) Format() string {
	return i.format
}

func (i *Image) SetFormat(format string) error {
	if i.Released() {
		return ErrEngineReleased
	}
	if err := i.mw.SetImageFormat(format); err != nil {
		return err
	}
	if err := i.sync(); err != nil {
		return err
	}
	return nil
}

func (i *Image) Released() bool {
	return i.mw == nil
}

func (i *Image) Release() {
	if i.mw != nil {
		i.mw.Destroy()
		i.mw = nil
	}
}

func (i *Image) Clone() imgry.Image {
	i2 := &Image{}
	i2.data = i.data
	i2.width = i.width
	i2.height = i.height
	i2.format = i.format
	if i.mw != nil && i.mw.IsVerified() {
		i2.mw = i.mw.Clone()
	}
	return i2
}

func (i *Image) SizeIt(sz *imgry.Sizing) error {
	if i.Released() {
		return ErrEngineReleased
	}

	if err := i.sizeFrames(sz); err != nil {
		return err
	}

	if sz.Format != "" {
		if err := i.mw.SetFormat(sz.Format); err != nil {
			return err
		}
	}

	if sz.Quality > 0 {
		if err := i.mw.SetImageCompressionQuality(uint(sz.Quality)); err != nil {
			return err
		}
	}

	if err := i.sync(sz.Flatten); err != nil {
		return err
	}

	return nil
}

func (i *Image) sizeFrames(sz *imgry.Sizing) error {
	var canvas *imagick.MagickWand
	var bg *imagick.PixelWand

	// Shortcut if there is nothing to size
	if sz.Size.Equal(imgry.ZeroRect) && sz.CropBox.Equal(imgry.ZeroFloatingRect) {
		return nil
	}

	coalesceAndDeconstruct := !sz.Flatten && i.mw.GetNumberImages() > 1
	if coalesceAndDeconstruct {
		i.mw = i.mw.CoalesceImages()
	}

	if sz.Canvas != nil {
		// If the user requested a canvas.
		canvas = imagick.NewMagickWand()
		bg = imagick.NewPixelWand()
		bg.SetColor("transparent")

		defer func() {
			bg.Destroy()
			if canvas != nil && canvas != i.mw {
				canvas.Destroy()
			}
		}()
	}

	i.mw.SetFirstIterator()
	for n := true; n; n = i.mw.NextImage() {
		pw, ph := int(i.mw.GetImageWidth()), int(i.mw.GetImageHeight())
		srcSize := imgry.NewRect(pw, ph)

		// Initial crop of the source image
		cropBox, cropOrigin, err := sz.CalcCropBox(srcSize)
		if err != nil {
			return err
		}

		if cropBox != nil && cropOrigin != nil && !cropBox.Equal(imgry.ZeroRect) {
			err := i.mw.CropImage(uint(cropBox.Width), uint(cropBox.Height), cropOrigin.X, cropOrigin.Y)
			if err != nil {
				return err
			}
			srcSize = cropBox
			i.mw.ResetImagePage("")
		}

		// Resize the image
		resizeRect, cropBox, cropOrigin := sz.CalcResizeRect(srcSize)
		if resizeRect != nil && !resizeRect.Equal(imgry.ZeroRect) {
			err := i.mw.ResizeImage(uint(resizeRect.Width), uint(resizeRect.Height), imagick.FILTER_LANCZOS, 1.0)
			if err != nil {
				return err
			}
			i.mw.ResetImagePage("")
		}

		// Perform any final crops from an operation
		if cropBox != nil && cropOrigin != nil && !cropBox.Equal(imgry.ZeroRect) {
			err := i.mw.CropImage(uint(cropBox.Width), uint(cropBox.Height), cropOrigin.X, cropOrigin.Y)
			if err != nil {
				return err
			}
			i.mw.ResetImagePage("")
		}

		// If we have a canvas we put the image at its center.
		if canvas != nil {
			canvas.NewImage(uint(sz.Canvas.Width), uint(sz.Canvas.Height), bg)
			canvas.SetImageBackgroundColor(bg)
			canvas.SetImageFormat(i.mw.GetImageFormat())

			x := (sz.Canvas.Width - int(i.mw.GetImageWidth())) / 2
			y := (sz.Canvas.Height - int(i.mw.GetImageHeight())) / 2
			canvas.CompositeImage(i.mw, imagick.COMPOSITE_OP_OVER, x, y)
			canvas.ResetImagePage("")
		}

		if sz.Flatten {
			break
		}
	}

	if canvas != nil {
		i.mw.Destroy()
		i.mw = canvas
	}

	if coalesceAndDeconstruct {
		// Compares each frame of the image, removes pixels that are already on the
		// background and updates offsets accordingly.
		i.mw = i.mw.DeconstructImages()
	}

	return nil
}

func (i *Image) WriteToFile(fn string) error {
	err := ioutil.WriteFile(fn, i.Data(), 0664)
	return err
}

func (i *Image) sync(optFlatten ...bool) error {
	if i.Released() {
		return ErrEngineReleased
	}

	var flatten bool
	if len(optFlatten) > 0 {
		flatten = optFlatten[0]
	}

	if flatten {
		i.data = i.mw.GetImageBlob()
	} else {
		i.data = i.mw.GetImagesBlob()
	}

	i.width = int(i.mw.GetImageWidth())
	i.height = int(i.mw.GetImageHeight())

	i.format = strings.ToLower(i.mw.GetImageFormat())
	if i.format == "jpeg" {
		i.format = "jpg"
	}

	return nil
}
