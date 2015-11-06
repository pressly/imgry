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

	assert.True(t, len(img.Data()) < origSize, fmt.Sprintf("Expecting %d < %d.", len(img.Data()), origSize))

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
