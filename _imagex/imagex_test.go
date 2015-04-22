package imagex

import (
	"io/ioutil"
	"log"
	"testing"
)

/*
TODO:
=====
1. Add test images for: jpeg, jpeg (cmyk), png, gif, gif (animated), ico, bmp
*/

func TestBasic(t *testing.T) {
	tdImage1, err := ioutil.ReadFile("/Users/peter/Dev/go/src/github.com/pressly/imgry/testdata/favicon.ico")
	if err != nil {
		t.Error(err)
	}

	ng := Engine{}

	im, err := ng.LoadBlob(tdImage1)
	if err != nil {
		t.Fatal(err)
	}

	log.Println(im.Width())
	log.Println(im.Height())
	log.Println(im.Format())

	// im2 := im.Clone()

	// sz := imgry.Sizing{Box: "300x"} // Box.. Rect.. Size.. ?
	// sz, _ := imgry.NewSizingFromQuery("size=800x")

	// err = im.SizeIt(sz)
	// if err != nil {
	//   t.Fatal(err)
	// }

	// defer im.Release()
	// // defer im2.Release()

	// im.WriteToFile("/tmp/image1.jpg")
}
