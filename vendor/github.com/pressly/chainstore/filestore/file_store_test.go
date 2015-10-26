package filestore_test

import (
	"testing"

	"github.com/pressly/chainstore"
	"github.com/pressly/chainstore/filestore"
	. "github.com/smartystreets/goconvey/convey"
)

func TestFileStore(t *testing.T) {
	var store chainstore.Store
	var err error

	Convey("Fsdb Open", t, func() {
		store = filestore.New(chainstore.TempDir(), 0755)
		err = nil
		So(err, ShouldEqual, nil)

		Convey("Put/Get/Del basic data", func() {
			err = store.Put("test.txt", []byte{1, 2, 3, 4})
			So(err, ShouldEqual, nil)

			data, err := store.Get("test.txt")
			So(err, ShouldEqual, nil)
			So(data, ShouldResemble, []byte{1, 2, 3, 4})
		})

		Convey("Auto-creating directories on put", func() {
			err = store.Put("hello/there/everyone.txt", []byte{1, 2, 3, 4})
			So(err, ShouldEqual, nil)
		})

	})
}
