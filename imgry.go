package imgry

import "errors"

const (
	VERSION = "1.0.0"
)

var (
	ErrInvalidImageData = errors.New("invalid image data")
)

type Engine interface {
	Version() string
	Initialize(tmpDir string) error
	Terminate()

	LoadFile(filename string, srcFormat ...string) (Image, error)
	LoadBlob(b []byte, srcFormat ...string) (Image, error)
	GetImageInfo(b []byte, srcFormat ...string) (*ImageInfo, error)
}

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
