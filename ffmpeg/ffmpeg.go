package ffmpeg

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
)

// Convert wraps the ffmpeg command and can be used to convert media files.
//   ffmpeg -i src -y dst
func Convert(src string, dst string) (err error) {
	if src, err = filepath.Abs(src); err != nil {
		return
	}

	if dst, err = filepath.Abs(dst); err != nil {
		return
	}

	if _, err = os.Stat(src); err != nil {
		return fmt.Errorf("Failed to open file %s: %q", src, err)
	}

	if _, err = os.Stat(path.Dir(dst)); err != nil {
		return fmt.Errorf("No such directory %s: %q", path.Dir(src), err)
	}

	cmd := exec.Command("ffmpeg", "-i", src, "-y", dst)

	var stderr io.ReadCloser
	if stderr, err = cmd.StderrPipe(); err != nil {
		return
	}

	if err = cmd.Run(); err != nil {
		return
	}

	errmsg, _ := ioutil.ReadAll(stderr)
	errstr := string(errmsg)
	if errstr != "" {
		return errors.New(errstr)
	}

	return nil
}
