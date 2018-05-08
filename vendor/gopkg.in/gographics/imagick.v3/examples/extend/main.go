// Port of http://members.shaw.ca/el.supremo/MagickWand/extent.htm to Go
package main

import (
	"os"

	"gopkg.in/gographics/imagick.v3/imagick"
)

func main() {
	imagick.Initialize()
	defer imagick.Terminate()

	var err error

	mw := imagick.NewMagickWand()
	pw := imagick.NewPixelWand()
	pw.SetColor("blue")

	err = mw.ReadImage("logo:")
	if err != nil {
		panic(err)
	}

	w := int(mw.GetImageWidth())
	h := int(mw.GetImageHeight())
	mw.SetImageBackgroundColor(pw)

	// This centres the original image on the new canvas.
	// Note that the extent's offset is relative to the
	// top left corner of the *original* image, so adding an extent
	// around it means that the offset will be negative
	err = mw.ExtentImage(1024, 768, -(1024-w)/2, -(768-h)/2)
	if err != nil {
		panic(err)
	}

	// Set the compression quality to 95 (high quality = low compression)
	err = mw.SetImageCompressionQuality(95)
	if err != nil {
		panic(err)
	}

	mw.DisplayImage(os.Getenv("DISPLAY"))
	if err != nil {
		panic(err)
	}
}
