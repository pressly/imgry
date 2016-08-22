package imgry

import (
	"errors"
	"fmt"
	"image"
	"net/url"
	"strconv"
	"strings"

	"math"
)

var (
	ZeroSizing = &Sizing{}

	DefaultSizingGranularity = 10
)

const (
	canvasMaxSize int = 1024
)

type Sizing struct {
	Size       *Rect         // The asking image size
	CropBox    *FloatingRect // The asking image crop box (as percentages)
	FocalPoint *FloatPoint   // The asking image focal point (as percentages)
	Canvas     *Rect

	Op          string
	Format      string
	Quality     int
	Granularity int
	Flatten     bool
}

func NewSizing() *Sizing {
	sz := &Sizing{}
	sz.Size = &Rect{}
	sz.CropBox = &FloatingRect{&FloatPoint{}, &FloatPoint{}}
	sz.FocalPoint = &FloatPoint{}
	sz.Granularity = DefaultSizingGranularity
	sz.Quality = 75
	sz.Flatten = false
	return sz
}

func NewSizingFromQuery(q string) (*Sizing, error) {
	sz := NewSizing()
	if err := sz.SetFromQuery(q); err != nil {
		return nil, err
	}
	return sz, nil
}

func (sz *Sizing) CalcCropBox(srcSize *Rect) (cropBox *Rect, cropOrigin *image.Point, err error) {
	x1, y1 := sz.CropBox.Min.X, sz.CropBox.Min.Y
	x2, y2 := sz.CropBox.Max.X, sz.CropBox.Max.Y
	srcW, srcH := float64(srcSize.Width), float64(srcSize.Height)

	ltXi := round(x1 * srcW)
	ltYi := round(y1 * srcH)
	rbXi := round(x2 * srcW)
	rbYi := round(y2 * srcH)

	if rbXi < ltXi || rbYi < ltYi {
		return nil, nil, errors.New("invalid box query param")
	}
	return &Rect{rbXi - ltXi, rbYi - ltYi}, &image.Point{ltXi, ltYi}, nil
}

func (sz *Sizing) CalcResizeRect(srcSize *Rect) (resizedRect *Rect, cropRect *Rect, cropOrigin *image.Point) {
	switch sz.Op {
	case "exact":
		resizedRect, cropRect, cropOrigin = sz.exactOp(srcSize)
	case "contain":
		resizedRect, cropRect, cropOrigin = sz.containOp(srcSize)
	case "contain2":
		resizedRect, cropRect, cropOrigin = sz.contain2Op(srcSize)
	case "expand":
		resizedRect, cropRect, cropOrigin = sz.expandOp(srcSize)
	case "cover":
		resizedRect, cropRect, cropOrigin = sz.coverOp(srcSize)
	case "balance":
		resizedRect, cropRect, cropOrigin = sz.balanceOp(srcSize)
	case "fitted":
		resizedRect, cropRect, cropOrigin = sz.fitted(srcSize)
	default:
		resizedRect, cropRect, cropOrigin = sz.exactOp(srcSize)
	}

	// Catch cropPoints that don't exist within the bounds of the image
	if cropOrigin != nil {
		negativePoint := cropOrigin.X < 0 || cropOrigin.Y < 0
		// check against the resized image from where we are actually going to
		// crop.
		oversizedPoint := cropOrigin.X > resizedRect.Width || cropOrigin.Y > resizedRect.Height
		if negativePoint || oversizedPoint {
			cropOrigin = &image.Point{}
		}
	}
	return resizedRect, cropRect, cropOrigin
}

func (sz *Sizing) exactOp(srcSize *Rect) (*Rect, *Rect, *image.Point) {
	return sz.calcScaledSize(srcSize, false), nil, nil
}

func (sz *Sizing) containOp(srcSize *Rect) (*Rect, *Rect, *image.Point) {
	rr := sz.calcScaledSize(srcSize, false)
	if srcSize.Width > rr.Width || srcSize.Height > rr.Height {
		return sz.contain2Op(srcSize)
	}
	return srcSize, nil, nil
}

func (sz *Sizing) contain2Op(srcSize *Rect) (*Rect, *Rect, *image.Point) {
	return sz.calcScaledSize(srcSize, true), nil, nil
}

func (sz *Sizing) fitted(srcSize *Rect) (*Rect, *Rect, *image.Point) {
	size := sz.calcScaledSize(srcSize, false)
	if sz.Canvas != nil {
		ratio := math.Min(float64(sz.Canvas.Width)/float64(size.Width), float64(sz.Canvas.Height)/float64(size.Height))
		if ratio < 1 {
			// This means the canvas is smaller than the source image.
			size.Width = int(float64(size.Width) * ratio)
			size.Height = int(float64(size.Height) * ratio)
		}
	}
	return size, nil, nil
}

func (sz *Sizing) expandOp(srcSize *Rect) (*Rect, *Rect, *image.Point) {
	rr := sz.calcScaledSize(srcSize, false)
	if srcSize.Width < rr.Width || srcSize.Height < rr.Height {
		return sz.calcScaledSize(srcSize, true), nil, nil
	}
	return srcSize, nil, nil
}

func (sz *Sizing) coverOp(srcSize *Rect) (*Rect, *Rect, *image.Point) {
	if sz.FocalPoint.Equal(ZeroFloatPoint) {
		sz.FocalPoint = NewFloatPoint(0.5, 0.5)
	}
	return sz.cropByOffset(srcSize)
}

func (sz *Sizing) balanceOp(srcSize *Rect) (*Rect, *Rect, *image.Point) {
	if sz.FocalPoint.Equal(ZeroFloatPoint) {
		sz.FocalPoint = NewFloatPoint(0.5, 0.33)
	}
	return sz.cropByOffset(srcSize)
}

func (sz *Sizing) cropByOffset(srcSize *Rect) (*Rect, *Rect, *image.Point) {
	rr := sz.calcScaledSize(srcSize, false)
	if rr.AspectRatio() < srcSize.AspectRatio() {
		return sz.cropByHeight(srcSize)
	} else {
		return sz.cropByWidth(srcSize)
	}
}

func (sz *Sizing) cropByWidth(srcSize *Rect) (*Rect, *Rect, *image.Point) {
	rr := sz.scaleToWidth(srcSize)
	diffY := float64(rr.Height)*sz.FocalPoint.Y - float64(sz.Size.Height)*0.5
	if diffY < 0 {
		diffY = 0
	}
	// y can possibly be negative or larger than the image
	y := min(round(diffY), rr.Height-sz.Size.Height)
	return rr, sz.Size, &image.Point{0, y} // TODO: hmm.. returning sz.Size here...??
}

func (sz *Sizing) cropByHeight(srcSize *Rect) (*Rect, *Rect, *image.Point) {
	rr := sz.scaleToHeight(srcSize)
	diffX := float64(rr.Width)*sz.FocalPoint.X - float64(sz.Size.Width)*0.5
	if diffX < 0 {
		diffX = 0
	}
	// x can possibly be negative or larger than the image
	x := min(round(diffX), rr.Width-sz.Size.Width)
	return rr, sz.Size, &image.Point{x, 0} // TODO: hmm.. returning sz.Size ..?
}

func (sz *Sizing) calcScaledSize(srcSize *Rect, scaleOrNot bool) *Rect {
	if sz.Size.Width == 0 {
		return sz.scaleToHeight(srcSize)
	}
	if sz.Size.Height == 0 {
		return sz.scaleToWidth(srcSize)
	}

	if scaleOrNot {
		if sz.Size.AspectRatio() > srcSize.AspectRatio() {
			return sz.scaleToHeight(srcSize)
		} else {
			return sz.scaleToWidth(srcSize)
		}
	} else {
		return sz.GranularizedSize()
	}
}

// Returns a granularized width and a height scaled to the same ratio as original size
func (sz *Sizing) scaleToWidth(srcSize *Rect) *Rect {
	r := &Rect{}
	r.Width = sz.GranularizedWidth()
	r.Height = round(float64(r.Width) / srcSize.AspectRatio())
	return r
}

// Returns a granularized height and a width scaled to the same ratio as original size
func (sz *Sizing) scaleToHeight(srcSize *Rect) *Rect {
	r := &Rect{}
	r.Height = sz.GranularizedHeight()
	r.Width = round(float64(r.Height) * srcSize.AspectRatio())
	return r
}

// Returns the granularized width
func (sz *Sizing) GranularizedWidth() int {
	return sz.granularize(sz.Size.Width)
}

// Returns the granularized asking height
func (sz *Sizing) GranularizedHeight() int {
	return sz.granularize(sz.Size.Height)
}

func (sz *Sizing) GranularizedSize() *Rect {
	return NewRect(sz.GranularizedWidth(), sz.GranularizedHeight())
}

// Returns the length rounded to the nearest multiple of the granularity
// For example a length of 83 and a granularity of 5 would return 85
func (sz *Sizing) granularize(length int) int {
	rem := round(math.Mod(float64(length), float64(sz.Granularity)))
	factor := length/sz.Granularity + rem*2/sz.Granularity
	return sz.Granularity * factor
}

func (sz *Sizing) SetFromQuery(q string) error {
	var err error

	if q == "" {
		return fmt.Errorf("no query given")
	}

	query, err := url.ParseQuery(q)
	if err != nil {
		return err
	}

	// Resize size
	size := query.Get("size")
	if size == "" {
		size = query.Get("s")
	}
	if size != "" && size != "x" {
		sz.Size, err = NewRectFromQuery(size)
		if err != nil {
			return err
		}
	}

	// Canvas size
	canvas := query.Get("canvas")
	if canvas != "" && canvas != "x" {
		sz.Canvas, err = NewRectFromQuery(canvas)
		if err != nil {
			return err
		}
		sz.Canvas.Width = min(sz.Canvas.Width, canvasMaxSize)
		sz.Canvas.Height = min(sz.Canvas.Height, canvasMaxSize)
	}

	// Sizing operation
	sz.Op = query.Get("op")

	// Quality
	sz.Quality = 75
	if query.Get("hq") == "" {
		if query.Get("q") != "" {
			sz.Quality, err = strconv.Atoi(query.Get("q"))
			if err != nil {
				return err
			}
		}
	} else {
		sz.Quality = 0 // 0 is we don't adjust the quality, presuming 100%
	}

	// FocalPoint
	fp := query.Get("fp")
	if fp == "" {
		fp = query.Get("focal")
	}
	if fp != "" {
		sz.FocalPoint, err = NewFloatPointFromQuery(fp)
		if err != nil {
			return err
		}
	}

	// CropBox
	cb := query.Get("cb")
	if cb == "" {
		cb = query.Get("box")
	}
	if cb != "" {
		sz.CropBox, err = NewFloatingRectFromQuery(cb)
		if err != nil {
			return err
		}
	}

	// Format
	sz.Format = query.Get("format")

	// Granularity
	g := query.Get("g")
	if g != "" {
		sz.Granularity, err = strconv.Atoi(g)
		if err != nil {
			return err
		}
		if sz.Granularity <= 0 {
			sz.Granularity = DefaultSizingGranularity
		}
	}

	// Flatten
	if query.Get("flatten") != "" {
		sz.Flatten = true
	}

	return nil
}

func (sz *Sizing) ToQuery() url.Values {
	u := url.Values{}

	if !sz.Size.Equal(ZeroRect) {
		u.Add("s", sz.Size.ToString())
	}
	if sz.Canvas != nil {
		u.Add("canvas", sz.Canvas.ToString())
	}
	if sz.Op != "" {
		u.Add("op", sz.Op)
	}
	if sz.Quality != 0 {
		u.Add("q", strconv.Itoa(sz.Quality))
	}
	if !sz.FocalPoint.Equal(ZeroFloatPoint) {
		u.Add("fp", sz.FocalPoint.ToString())
	}
	if !sz.CropBox.Equal(ZeroFloatingRect) {
		u.Add("cb", sz.CropBox.ToString())
	}
	if sz.Format != "" {
		u.Add("format", sz.Format)
	}
	if sz.Granularity > 1 {
		u.Add("g", strconv.Itoa(sz.Granularity))
	}
	if sz.Flatten {
		u.Add("flatten", "1")
	}

	return u
}

var (
	ZeroRect         = &Rect{}
	ZeroFloatingRect = &FloatingRect{&FloatPoint{}, &FloatPoint{}}
	ZeroFloatPoint   = &FloatPoint{}
)

type Rect struct {
	Width, Height int
}

func NewRect(w, h int) *Rect {
	return &Rect{Width: w, Height: h}
}

func NewRectFromQuery(q string) (*Rect, error) {
	if q == "" {
		return NewRect(0, 0), nil
	}

	wh := strings.Split(q, "x")
	if len(wh) != 2 {
		return nil, fmt.Errorf("invalid rect query: %s", q)
	}

	var w, h int

	if wh[0] != "" {
		fw, err := strconv.ParseFloat(wh[0], 64)
		if err != nil {
			return nil, err
		}
		w = int(fw)
	}
	if wh[1] != "" {
		fh, err := strconv.ParseFloat(wh[1], 64)
		if err != nil {
			return nil, err
		}
		h = int(fh)
	}
	return &Rect{w, h}, nil
}

func (r *Rect) AspectRatio() float64 {
	return float64(r.Width) / float64(r.Height)
}

func (r *Rect) Equal(other *Rect) bool {
	return (r.Width == other.Width) && (r.Height == other.Height)
}

// Return the width and height difference with the src rect
func (r *Rect) DiffSize(src *Rect) (int, int) {
	return src.Width - r.Width, src.Height - r.Height
}

func (r *Rect) ToString() string {
	return fmt.Sprintf("%dx%d", r.Width, r.Height)
}

type FloatingRect struct {
	Min, Max *FloatPoint // can be whole or percentages
}

func NewFloatingRect(x0, y0, x1, y2 float64) *FloatingRect {
	return &FloatingRect{&FloatPoint{x0, y0}, &FloatPoint{x1, y2}}
}

func NewFloatingRectFromQuery(q string) (*FloatingRect, error) {
	if q == "" {
		return NewFloatingRect(0, 0, 0, 0), nil
	}

	mm := strings.Split(q, ",")
	if len(mm) != 4 {
		return nil, fmt.Errorf("invalid floating rect query:", q)
	}

	var err error
	var min, max *FloatPoint
	min, err = NewFloatPointFromQuery(fmt.Sprintf("%s,%s", mm[0], mm[1]))
	if err != nil {
		return nil, err
	}
	max, err = NewFloatPointFromQuery(fmt.Sprintf("%s,%s", mm[2], mm[3]))
	if err != nil {
		return nil, err
	}
	return &FloatingRect{min, max}, nil
}

func (f *FloatingRect) Equal(other *FloatingRect) bool {
	return f.Min.Equal(other.Min) && f.Max.Equal(other.Max)
}

func (f *FloatingRect) ToString() string {
	return fmt.Sprintf("%s,%s", f.Min.ToString(), f.Max.ToString())
}

type FloatPoint struct {
	X, Y float64
}

func NewFloatPoint(x, y float64) *FloatPoint {
	return &FloatPoint{x, y}
}

func NewFloatPointFromQuery(q string) (*FloatPoint, error) {
	if q == "" {
		return NewFloatPoint(0, 0), nil
	}

	xy := strings.Split(q, ",")
	if len(xy) != 2 {
		return nil, fmt.Errorf("invalid focal point query: %s", q)
	}

	var err error
	var x, y float64
	x, err = strconv.ParseFloat(xy[0], 64)
	if err != nil {
		return nil, err
	}
	y, err = strconv.ParseFloat(xy[1], 64)
	if err != nil {
		return nil, err
	}

	// TODO: remove the below conversion, unfortunately it resides because a previous
	// interface of the imgry API used whole numbers for percentages for focal points
	// and decimals for the cropbox. Make them consistent and remove the below.
	if x > 1 {
		x = x / 100.0
	}
	if y > 1 {
		y = y / 100.0
	}
	return &FloatPoint{x, y}, nil
}

// Returns whether the two FloatPoints have equal values
func (f *FloatPoint) Equal(other *FloatPoint) bool {
	return (f.X == other.X) && (f.Y == other.Y)
}

// Returns a comma delimited string representing the point (ie. "0.1,0.2" or "100,150")
func (f *FloatPoint) ToString() string {
	if f.X < 1.0 || f.Y < 1.0 {
		return fmt.Sprintf("%1.2f,%1.2f", f.X, f.Y) // Percentages
	}
	return fmt.Sprintf("%1.0f,%1.0f", f.X, f.Y) // Whole
}

// Rounding function for float64 numbers
func round(in float64) int {
	if in < 0 {
		return int(math.Ceil(in - 0.5))
	} else {
		return int(math.Floor(in + 0.5))
	}
}

// Min function for ints
func min(first, second int) int {
	return int(math.Min(float64(first), float64(second)))
}
