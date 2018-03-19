package s3store

import (
	"bytes"
	"errors"
)

type readSeeker struct {
	buffer *bytes.Buffer
	index  int64
}

func newReadSeeker(v []byte) *readSeeker {
	return &readSeeker{buffer: bytes.NewBuffer(v), index: 0}
}

func (rs *readSeeker) Bytes() []byte {
	return rs.buffer.Bytes()
}

func (rs *readSeeker) Read(p []byte) (int, error) {
	n, err := bytes.NewBuffer(rs.buffer.Bytes()[rs.index:]).Read(p)

	if err == nil {
		if rs.index+int64(len(p)) < int64(rs.buffer.Len()) {
			rs.index += int64(len(p))
		} else {
			rs.index = int64(rs.buffer.Len())
		}
	}

	return n, err
}
func (rs *readSeeker) Seek(offset int64, whence int) (int64, error) {
	var err error
	var index int64 = 0

	switch whence {
	case 0:
		if offset >= int64(rs.buffer.Len()) || offset < 0 {
			err = errors.New("Invalid Offset.")
		} else {
			rs.index = offset
			index = offset
		}
	default:
		err = errors.New("Unsupported Seek Method.")
	}

	return index, err
}
