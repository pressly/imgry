package imagick

import (
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
