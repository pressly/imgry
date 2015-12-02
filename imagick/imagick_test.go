package imagick

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/pressly/imgry"
)

func TestLoadBlob(t *testing.T) {
	tdImage1, err := ioutil.ReadFile("../testdata/image1.jpg")
	assert.NoError(t, err)

	ng := Engine{}
	im, err := ng.LoadBlob(tdImage1)
	assert.NoError(t, err)
	defer im.Release()

	sz, _ := imgry.NewSizingFromQuery("size=800x")
	err = im.SizeIt(sz)
	assert.NoError(t, err)

	im2Path := "/tmp/image1.jpg"
	im.WriteToFile(im2Path)

	im2, err := ng.LoadFile(im2Path)
	assert.NoError(t, err)

	assert.True(t, im2.Width() == 800)
	assert.Equal(t, "jpg", im2.Format())

	err = im2.SetFormat("png")
	assert.NoError(t, err)
}

func TestGetImageInfo(t *testing.T) {
	tdImage1, err := ioutil.ReadFile("../testdata/image1.jpg")
	assert.NoError(t, err)

	ng := Engine{}
	imfo, err := ng.GetImageInfo(tdImage1)
	assert.NoError(t, err)

	assert.Equal(t, imfo.Width, 1600)
	assert.Equal(t, imfo.Height, 1200)
	assert.True(t, float64(int(imfo.AspectRatio*1000))/1000 == 1.333)
	assert.True(t, imfo.ContentLength == 451317)
}

func TestIssue8GIFResize(t *testing.T) {
	var sz *imgry.Sizing
	var img imgry.Image
	var err error

	ng := Engine{}

	img, err = ng.LoadFile("../testdata/issue-8.gif")
	assert.NoError(t, err)

	assert.Equal(t, 131, img.Width())
	assert.Equal(t, 133, img.Height())

	origSize := len(img.Data())
	assert.Equal(t, 393324, origSize)

	img.Release()

	// Resizing to 750, which is slightly smaller.
	img, err = ng.LoadFile("../testdata/issue-8.gif")
	assert.NoError(t, err)

	sz, _ = imgry.NewSizingFromQuery("size=750x")
	err = img.SizeIt(sz)
	assert.NoError(t, err)

	assert.Equal(t, 750, img.Width())
	assert.Equal(t, 422, img.Height())

	// We should be able to expect this someday, but for now it seems like the
	// number of colors after resizing affect the size of the file.
	//
	// See http://www.imagemagick.org/discourse-server/viewtopic.php?t=22505#p93859
	//assert.True(t, len(img.Data()) < origSize, fmt.Sprintf("Expecting %d < %d.", len(img.Data()), origSize))

	err = img.WriteToFile("../testdata/issue-8.700.gif")
	assert.NoError(t, err)

	img.Release()

	// Resizing to 500, which is smaller.
	img, err = ng.LoadFile("../testdata/issue-8.gif")
	assert.NoError(t, err)

	sz, _ = imgry.NewSizingFromQuery("size=500x")
	err = img.SizeIt(sz)
	assert.NoError(t, err)

	assert.Equal(t, 500, img.Width())
	assert.Equal(t, 282, img.Height())

	assert.True(t, len(img.Data()) < origSize, fmt.Sprintf("Expecting %d < %d.", len(img.Data()), origSize))

	err = img.WriteToFile("../testdata/issue-8.500.gif")
	assert.NoError(t, err)

	img.Release()

	// Resizing to 900, which is larger.
	img, err = ng.LoadFile("../testdata/issue-8.gif")
	assert.NoError(t, err)

	sz, _ = imgry.NewSizingFromQuery("size=900x")
	err = img.SizeIt(sz)
	assert.NoError(t, err)

	assert.Equal(t, 900, img.Width())
	assert.Equal(t, 507, img.Height())

	assert.True(t, len(img.Data()) > origSize, fmt.Sprintf("Expecting %d > %d.", len(img.Data()), origSize))

	err = img.WriteToFile("../testdata/issue-8.900.gif")
	assert.NoError(t, err)

	img.Release()

	// Resizing to 200, which is smaller.
	img, err = ng.LoadFile("../testdata/issue-8.gif")
	assert.NoError(t, err)

	sz, _ = imgry.NewSizingFromQuery("size=200x")
	err = img.SizeIt(sz)
	assert.NoError(t, err)

	assert.Equal(t, 200, img.Width())
	assert.Equal(t, 113, img.Height())

	assert.True(t, len(img.Data()) < origSize, fmt.Sprintf("Expecting %d < %d.", len(img.Data()), origSize))

	err = img.WriteToFile("../testdata/issue-8.200.gif")
	assert.NoError(t, err)

	img.Release()

	// Resizing to 150, which is smaller.
	img, err = ng.LoadFile("../testdata/issue-8.gif")
	assert.NoError(t, err)

	sz, _ = imgry.NewSizingFromQuery("size=150x")
	err = img.SizeIt(sz)
	assert.NoError(t, err)

	assert.Equal(t, 150, img.Width())
	assert.Equal(t, 84, img.Height())

	assert.True(t, len(img.Data()) < origSize, fmt.Sprintf("Expecting %d < %d.", len(img.Data()), origSize))

	err = img.WriteToFile("../testdata/issue-8.150.gif")
	assert.NoError(t, err)

	img.Release()
}

func TestIssue10OpFittedJPEG(t *testing.T) {

	portrait := func(fn func(img imgry.Image) error) {
		var img imgry.Image
		var err error
		ng := Engine{}

		img, err = ng.LoadFile("../testdata/issue-10-p.jpg")
		assert.NoError(t, err)

		assert.Equal(t, 150, img.Width())
		assert.Equal(t, 330, img.Height())

		err = fn(img)
		assert.NoError(t, err)

		img.Release()
	}

	// Smaller canvas
	portrait(func(img imgry.Image) (err error) {
		// Note that we scale the image to 200 first.
		sz, _ := imgry.NewSizingFromQuery("size=200x&canvas=320x200&op=fitted")
		err = img.SizeIt(sz)
		assert.NoError(t, err)

		assert.Equal(t, 320, img.Width())
		assert.Equal(t, 200, img.Height())

		err = img.WriteToFile("../testdata/issue-10-p-200x-320x200.jpg")
		assert.NoError(t, err)

		return
	})

	// Larger canvas
	portrait(func(img imgry.Image) (err error) {
		sz, _ := imgry.NewSizingFromQuery("size=150x&canvas=200x350&op=fitted")
		err = img.SizeIt(sz)
		assert.NoError(t, err)

		assert.Equal(t, 200, img.Width())
		assert.Equal(t, 350, img.Height())

		err = img.WriteToFile("../testdata/issue-10-p-150x-200x350.jpg")
		assert.NoError(t, err)

		return
	})

	landscape := func(fn func(img imgry.Image) error) {
		var img imgry.Image
		var err error
		ng := Engine{}

		img, err = ng.LoadFile("../testdata/issue-10-l.jpg")
		assert.NoError(t, err)

		assert.Equal(t, 320, img.Width())
		assert.Equal(t, 200, img.Height())

		err = fn(img)
		assert.NoError(t, err)

		img.Release()
	}

	// Smaller canvas
	landscape(func(img imgry.Image) (err error) {
		sz, _ := imgry.NewSizingFromQuery("size=200x&canvas=150x150&op=fitted")
		err = img.SizeIt(sz)
		assert.NoError(t, err)

		assert.Equal(t, 150, img.Width())
		assert.Equal(t, 150, img.Height())

		err = img.WriteToFile("../testdata/issue-10-l-200x-150x150.jpg")
		assert.NoError(t, err)

		return
	})

	// Larger canvas
	landscape(func(img imgry.Image) (err error) {
		sz, _ := imgry.NewSizingFromQuery("size=320x&canvas=380x340&op=fitted")
		err = img.SizeIt(sz)
		assert.NoError(t, err)

		assert.Equal(t, 380, img.Width())
		assert.Equal(t, 340, img.Height())

		err = img.WriteToFile("../testdata/issue-10-l-320x-380x340.jpg")
		assert.NoError(t, err)

		return
	})

}

func TestIssue10OpFittedPNG(t *testing.T) {

	testimage := func(fn func(img imgry.Image) error) {
		var img imgry.Image
		var err error
		ng := Engine{}

		img, err = ng.LoadFile("../testdata/issue-10-l.png")
		assert.NoError(t, err)

		assert.Equal(t, 600, img.Width())
		assert.Equal(t, 206, img.Height())

		err = fn(img)
		assert.NoError(t, err)

		img.Release()
	}

	// Smaller canvas
	testimage(func(img imgry.Image) (err error) {
		// Note that we scale the image to 200 first.
		sz, _ := imgry.NewSizingFromQuery("size=200x&canvas=150x120&op=fitted")
		err = img.SizeIt(sz)
		assert.NoError(t, err)

		assert.Equal(t, 150, img.Width())
		assert.Equal(t, 120, img.Height())

		err = img.WriteToFile("../testdata/issue-10-l-200x-150x120.png")
		assert.NoError(t, err)

		return
	})

	// Larger canvas
	testimage(func(img imgry.Image) (err error) {
		sz, _ := imgry.NewSizingFromQuery("size=600x&canvas=650x650&op=fitted")
		err = img.SizeIt(sz)
		assert.NoError(t, err)

		assert.Equal(t, 650, img.Width())
		assert.Equal(t, 650, img.Height())

		err = img.WriteToFile("../testdata/issue-10-l-600x-650x650.png")
		assert.NoError(t, err)

		return
	})

	// Larger resize
	testimage(func(img imgry.Image) (err error) {
		sz, _ := imgry.NewSizingFromQuery("size=800x&canvas=650x650&op=fitted")
		err = img.SizeIt(sz)
		assert.NoError(t, err)

		assert.Equal(t, 650, img.Width())
		assert.Equal(t, 650, img.Height())

		err = img.WriteToFile("../testdata/issue-10-l-800x-650x650.png")
		assert.NoError(t, err)

		return
	})

}

func TestIssue10OpFittedGIF(t *testing.T) {

	testimage := func(fn func(img imgry.Image) error) {
		var img imgry.Image
		var err error
		ng := Engine{}

		img, err = ng.LoadFile("../testdata/issue-10-p.gif")
		assert.NoError(t, err)

		assert.Equal(t, 48, img.Width())
		assert.Equal(t, 64, img.Height())

		err = fn(img)
		assert.NoError(t, err)

		img.Release()
	}

	// Smaller canvas
	testimage(func(img imgry.Image) (err error) {
		// Note that we scale the image to 200 first.
		sz, _ := imgry.NewSizingFromQuery("size=200x&canvas=150x120&op=fitted")
		err = img.SizeIt(sz)
		assert.NoError(t, err)

		assert.Equal(t, 150, img.Width())
		assert.Equal(t, 120, img.Height())

		err = img.WriteToFile("../testdata/issue-10-p-200x-150x120.gif")
		assert.NoError(t, err)

		return
	})

	// Larger canvas
	testimage(func(img imgry.Image) (err error) {
		sz, _ := imgry.NewSizingFromQuery("size=48x&canvas=100x100&op=fitted")
		err = img.SizeIt(sz)
		assert.NoError(t, err)

		assert.Equal(t, 100, img.Width())
		assert.Equal(t, 100, img.Height())

		err = img.WriteToFile("../testdata/issue-10-p-48x-100x100.gif")
		assert.NoError(t, err)

		return
	})

	// Larger resize
	testimage(func(img imgry.Image) (err error) {
		sz, _ := imgry.NewSizingFromQuery("size=100x&canvas=200x200&op=fitted")
		err = img.SizeIt(sz)
		assert.NoError(t, err)

		assert.Equal(t, 200, img.Width())
		assert.Equal(t, 200, img.Height())

		err = img.WriteToFile("../testdata/issue-10-l-100x-200x200.gif")
		assert.NoError(t, err)

		return
	})

}

func TestIssue10OpFittedTestForLeaks(t *testing.T) {

	testimage := func(fn func(img imgry.Image) error) {
		var img imgry.Image
		var err error
		ng := Engine{}

		img, err = ng.LoadFile("../testdata/issue-10-p.gif")
		assert.NoError(t, err)

		assert.Equal(t, 48, img.Width())
		assert.Equal(t, 64, img.Height())

		err = fn(img)
		assert.NoError(t, err)

		img.Release()
	}

	for i := 0; i < 200; i++ {

		testimage(func(img imgry.Image) (err error) {
			sz, _ := imgry.NewSizingFromQuery("size=200x&canvas=150x120&op=fitted")
			err = img.SizeIt(sz)
			assert.NoError(t, err)

			assert.Equal(t, 150, img.Width())
			assert.Equal(t, 120, img.Height())

			err = img.WriteToFile("../testdata/issue-10-p-leak.gif")
			assert.NoError(t, err)

			return
		})
	}

}

func TestIssue25(t *testing.T) {
	var img imgry.Image
	var err error
	ng := Engine{}

	img, err = ng.LoadFile("../testdata/issue-25.jpg")
	assert.NoError(t, err)

	assert.Equal(t, 1600, img.Width())
	assert.Equal(t, 480, img.Height())

	assert.NoError(t, err)

	sz, _ := imgry.NewSizingFromQuery("format=jpeg&size=750x922&op=cover")
	err = img.SizeIt(sz)
	assert.NoError(t, err)

	assert.Equal(t, 750, img.Width())
	assert.Equal(t, 920, img.Height())

	err = img.WriteToFile("../testdata/issue-25-out.jpg")
	assert.NoError(t, err)

	img.Release()
}
