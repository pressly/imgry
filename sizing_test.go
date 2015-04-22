package imgry

import (
	"image"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	origTestRect = NewRect(640, 480)
	testRect1    = NewRect(300, 300)
	testRect2    = NewRect(300, 200)
	testRect3    = NewRect(200, 300)
	testRect4    = NewRect(1000, 667)
)

// Geometry tests

func TestScaleSize500x500(t *testing.T) {
	sz, err := NewSizingFromQuery("s=500x500")
	assert.NoError(t, err)
	result := sz.calcScaledSize(testRect1, true)
	expected := NewRect(500, 500)
	assert.Equal(t, expected, result)
}

func TestScaleSizeFloats500x500(t *testing.T) {
	sz, err := NewSizingFromQuery("s=500.50x500.123")
	assert.NoError(t, err)
	result := sz.calcScaledSize(testRect1, true)
	expected := NewRect(500, 500)
	assert.Equal(t, expected, result)
}

func TestScaleSize200x200(t *testing.T) {
	sz, err := NewSizingFromQuery("s=200x200")
	assert.NoError(t, err)
	result := sz.calcScaledSize(testRect1, true)
	expected := NewRect(200, 200)
	assert.Equal(t, expected, result)
}

func TestScaleToHeightx150(t *testing.T) {
	sz, err := NewSizingFromQuery("s=x150&g=1")
	assert.NoError(t, err)
	result := sz.scaleToHeight(testRect2)
	expected := NewRect(225, 150)
	assert.Equal(t, expected, result)
}

func TestScaleToHeightx150HeightLarger(t *testing.T) {
	sz, err := NewSizingFromQuery("s=x150&g=1")
	assert.Nil(t, err)
	result := sz.scaleToHeight(testRect3)
	expected := NewRect(100, 150)
	assert.Equal(t, expected, result)
}

func TestScaleToWidth150xWidthLarger(t *testing.T) {
	sz, err := NewSizingFromQuery("s=150x&g=1")
	assert.Nil(t, err)
	result := sz.scaleToWidth(testRect2)
	expected := NewRect(150, 100)
	assert.Equal(t, expected, result)
}

func TestScaleToWidth150xHeightLarger(t *testing.T) {
	sz, err := NewSizingFromQuery("s=150x&g=1")
	assert.Nil(t, err)
	result := sz.scaleToWidth(testRect3)
	expected := NewRect(150, 225)
	assert.Equal(t, expected, result)
}

func TestScaleToWidth75x50(t *testing.T) {
	sz, err := NewSizingFromQuery("s=75x&g=1")
	assert.Nil(t, err)
	result := sz.scaleToWidth(testRect2)
	expected := NewRect(75, 50)
	assert.Equal(t, expected, result)
}

func TestScaleToWidth75x113(t *testing.T) {
	sz, err := NewSizingFromQuery("s=75x&g=1")
	assert.Nil(t, err)
	result := sz.scaleToWidth(testRect3)
	expected := NewRect(75, 113)
	assert.Equal(t, expected, result)
}

func TestScaleToWidth150x150(t *testing.T) {
	sz, err := NewSizingFromQuery("s=150x150&g=1")
	assert.Nil(t, err)
	result := sz.scaleToWidth(testRect2)
	expected := NewRect(150, 100)
	assert.Equal(t, expected, result)
}

// Operations Test

func TestGetCropRect(t *testing.T) {
	sz, err := NewSizingFromQuery("cb=0.1,0.1,0.9,0.9")
	assert.Nil(t, err)
	result, _, err := sz.CalcCropBox(origTestRect)
	assert.Nil(t, err)
	expected := NewRect(512, 384)
	assert.Equal(t, expected, result)
}

func TestExact320x240(t *testing.T) {
	sz, err := NewSizingFromQuery("s=320x240")
	assert.Nil(t, err)
	result, _, _ := sz.exactOp(origTestRect)
	expected := NewRect(320, 240)
	assert.Equal(t, expected, result)
}

func TestExact300x240(t *testing.T) {
	sz, err := NewSizingFromQuery("s=300x240")
	assert.Nil(t, err)
	result, _, _ := sz.exactOp(origTestRect)
	expected := NewRect(300, 240)
	assert.Equal(t, expected, result)
}

func TestExact320(t *testing.T) {
	sz, err := NewSizingFromQuery("s=320x")
	assert.Nil(t, err)
	result, _, _ := sz.exactOp(origTestRect)
	expected := NewRect(320, 240)
	assert.Equal(t, expected, result)
}

func TestContain(t *testing.T) {
	sz, err := NewSizingFromQuery("s=900x500")
	assert.Nil(t, err)
	result, _, _ := sz.containOp(origTestRect)
	expected := NewRect(640, 480)
	assert.Equal(t, expected, result)
}

func TestNotContain(t *testing.T) {
	sz, err := NewSizingFromQuery("s=320x300")
	assert.Nil(t, err)
	result, _, _ := sz.containOp(origTestRect)
	expected := NewRect(320, 240)
	assert.Equal(t, expected, result)
}

func TestContain2(t *testing.T) {
	sz, err := NewSizingFromQuery("s=900x500")
	assert.Nil(t, err)
	result, _, _ := sz.contain2Op(origTestRect)
	expected := NewRect(667, 500)
	assert.Equal(t, expected, result)
}

func TestNotContain2(t *testing.T) {
	sz, err := NewSizingFromQuery("s=320x300")
	assert.Nil(t, err)
	result, _, _ := sz.contain2Op(origTestRect)
	expected := NewRect(320, 240)
	assert.Equal(t, expected, result)
}

func TestExpand1(t *testing.T) {
	sz, err := NewSizingFromQuery("s=600x400")
	assert.Nil(t, err)
	result, _, _ := sz.expandOp(origTestRect)
	expected := NewRect(640, 480)
	assert.Equal(t, expected, result)
}

func TestExpand2(t *testing.T) {
	sz, err := NewSizingFromQuery("s=700x300")
	assert.Nil(t, err)
	result, _, _ := sz.expandOp(origTestRect)
	expected := NewRect(400, 300)
	assert.Equal(t, expected, result)
}

func TestExpand3(t *testing.T) {
	sz, err := NewSizingFromQuery("s=700x1000")
	assert.Nil(t, err)
	result, _, _ := sz.expandOp(origTestRect)
	expected := NewRect(700, 525)
	assert.Equal(t, expected, result)
}

func TestCover(t *testing.T) {
	sz, err := NewSizingFromQuery("s=300x200")
	assert.Nil(t, err)
	resize, result, point := sz.coverOp(origTestRect)
	assert.NotNil(t, resize)
	assert.NotNil(t, point)
	expected := NewRect(300, 200)
	assert.Equal(t, expected, result)
}

func TestCover2(t *testing.T) {
	sz, err := NewSizingFromQuery("s=200x300")
	assert.Nil(t, err)
	resize, result, point := sz.coverOp(origTestRect)
	assert.NotNil(t, resize)
	assert.NotNil(t, point)
	expected := NewRect(200, 300)
	assert.Equal(t, expected, result)
}

func TestCoverWithFocusPoint(t *testing.T) {
	sz, err := NewSizingFromQuery("s=300x200&fp=10,0")
	assert.Nil(t, err)
	resize, result, point := sz.coverOp(origTestRect)
	assert.NotNil(t, resize)
	assert.NotNil(t, point)
	expected := NewRect(300, 200)
	assert.Equal(t, expected, result)
}

func TestCoverWithCrop(t *testing.T) {
	sz, err := NewSizingFromQuery("s=100x100&cb=0.5,0.0,1,1")
	box, point, err := sz.CalcCropBox(origTestRect)
	assert.NotNil(t, point)
	assert.Nil(t, err)
	expected := NewRect(320, 480)
	assert.Equal(t, expected, box)
	resize, box, point := sz.coverOp(box)
	assert.NotNil(t, resize)
	assert.NotNil(t, box)
	assert.NotNil(t, point)
	assert.Equal(t, NewRect(100, 150), resize)
}

func TestBalance(t *testing.T) {
	s, err := NewSizingFromQuery("s=300x200")
	assert.Nil(t, err)
	resize, result, point := s.balanceOp(origTestRect)
	assert.NotNil(t, resize)
	assert.NotNil(t, point)
	expected := NewRect(300, 200)
	assert.Equal(t, expected, result)
}

func TestBalance2(t *testing.T) {
	sz, err := NewSizingFromQuery("s=200x300")
	assert.Nil(t, err)
	resize, result, point := sz.balanceOp(origTestRect)
	assert.NotNil(t, resize)
	assert.NotNil(t, point)
	expected := NewRect(200, 300)
	assert.Equal(t, expected, result)
}

func TestBalanceWithFocusPoint(t *testing.T) {
	sz, err := NewSizingFromQuery("s=300x200&fp=10,0")
	assert.Nil(t, err)
	resize, result, point := sz.balanceOp(origTestRect)
	assert.NotNil(t, resize)
	assert.NotNil(t, point)
	expected := NewRect(300, 200)
	assert.Equal(t, expected, result)
}

func TestOperateCalculatesNegativeFP(t *testing.T) {
	sz, err := NewSizingFromQuery("s=700x&op=cover")
	assert.Nil(t, err)
	resize, result, point := sz.CalcResizeRect(testRect4)
	assert.NotNil(t, resize)
	assert.Equal(t, &image.Point{}, point)
	expected := NewRect(700, 0)
	assert.Equal(t, expected, result)
}

func TestOperateCalculatesOversizedFP(t *testing.T) {
	sz, err := NewSizingFromQuery("s=x500&op=cover")
	assert.Nil(t, err)
	resize, result, point := sz.CalcResizeRect(testRect4)
	assert.NotNil(t, resize)
	assert.Equal(t, &image.Point{}, point)
	expected := NewRect(0, 500)
	assert.Equal(t, expected, result)
}

// Query Tests

func TestFromQuery(t *testing.T) {
	query := "size=100x200&focal=0.1,0.2&hq=1&op=contain2&g=13&box=0.1,0.1,0.8,0.8&format=png"
	sz, err := NewSizingFromQuery(query)
	assert.Nil(t, err)
	assert.True(t, sz.Size.Equal(&Rect{100, 200}))
	assert.True(t, sz.FocalPoint.Equal(&FloatPoint{0.1, 0.2}))
	assert.Equal(t, 0, sz.Quality)
	assert.Equal(t, "contain2", sz.Op)
	assert.Equal(t, 13, sz.Granularity)
	assert.True(t, sz.CropBox.Equal(&FloatingRect{&FloatPoint{0.1, 0.1}, &FloatPoint{0.8, 0.8}}))
	assert.Equal(t, "png", sz.Format)
}

func TestToQuery(t *testing.T) {
	sz := NewSizing()
	sz.Size = &Rect{100, 200}
	sz.FocalPoint = &FloatPoint{0.1, 0.2}
	sz.Quality = 90
	sz.Op = "contain2"
	sz.Granularity = 13
	sz.CropBox = &FloatingRect{&FloatPoint{0.1, 0.1}, &FloatPoint{0.8, 0.8}}
	sz.Format = "png"

	result := sz.ToQuery()
	expected, err := url.ParseQuery("s=100x200&fp=0.10,0.20&q=90&op=contain2&g=13&cb=0.10,0.10,0.80,0.80&format=png")
	assert.Nil(t, err)
	assert.Equal(t, expected, result)
}

func TestToQuery2(t *testing.T) {
	sz := NewSizing()
	sz.Size = &Rect{100, 200}
	sz.FocalPoint = &FloatPoint{30, 50}
	sz.Quality = 90
	sz.Op = "contain2"
	sz.Granularity = 13
	sz.CropBox = &FloatingRect{&FloatPoint{10, 10}, &FloatPoint{90, 90}}
	sz.Format = "png"

	result := sz.ToQuery()
	expected, err := url.ParseQuery("s=100x200&fp=30,50&q=90&op=contain2&g=13&cb=10,10,90,90&format=png")
	assert.Nil(t, err)
	assert.Equal(t, expected, result)
}
