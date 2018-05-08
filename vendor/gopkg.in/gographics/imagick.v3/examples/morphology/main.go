// Port of http://members.shaw.ca/el.supremo/MagickWand/grayscale.htm to Go
package main

import (
	"fmt"

	"gopkg.in/gographics/imagick.v3/imagick"
)

func image_blur() {
	mw := imagick.NewMagickWand()
	mw.SetSize(640, 480)
	mw.ReadImage("magick:logo")
	mw.SetFormat("png")

	kernel_info := imagick.NewKernelInfoBuiltIn(imagick.KERNEL_RING, "2,1")
	kernel_info.Scale(1.0, imagick.KERNEL_NORMALIZE_VALUE)
	kernel_values := kernel_info.ToArray()

	fmt.Println(kernel_values)
	mw.WriteImage("testLogoBefore.png")
	mw.MorphologyImage(imagick.MORPHOLOGY_CONVOLVE, 1, kernel_info)
	mw.WriteImage("testLogoBlurred.png")
}

func main() {
	imagick.Initialize()
	defer imagick.Terminate()
	image_blur()
}
