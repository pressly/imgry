package imgry

import "errors"

const (
	VERSION = "0.14.0"
)

var (
	ErrInvalidImageData = errors.New("invalid image data")
)

type Image interface {
	Data() []byte
	Width() int
	Height() int
	Format() string
	SetFormat(format string) error

	Release()
	Released() bool

	SizeIt(sizing *Sizing) error
	WriteToFile(string) error
}

type ImageInfo struct {
	URL           string  `json:"url"`
	Format        string  `json:"format"`
	Mimetype      string  `json:"mimetype"`
	Width         int     `json:"width"`
	Height        int     `json:"height"`
	AspectRatio   float64 `json:"aspect_ratio"`
	ContentLength int     `json:"content_length"`
}
