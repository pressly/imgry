package ffmpeg

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGIFToVideo(t *testing.T) {
	os.Remove("../testdata/issue-8.mp4")
	err := Convert("../testdata/issue-8.gif", "../testdata/issue-8.mp4")
	assert.NoError(t, err)
}
