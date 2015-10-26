package memstore_test

import (
	"testing"

	"github.com/pressly/chainstore"
	"github.com/pressly/chainstore/memstore"
	. "github.com/smartystreets/goconvey/convey"
)

func TestMemCacheStore(t *testing.T) {
	var store chainstore.Store
	store = memstore.New(10)

	Convey("MemCacheStore", t, func() {
		e := store.Put("hi", []byte{1, 2, 3})
		So(e, ShouldEqual, nil)

		obj, e := store.Get("hi")
		So(e, ShouldEqual, nil)
		So(obj, ShouldResemble, []byte{1, 2, 3})

		e = store.Put("bye", []byte{5, 6, 7, 8, 9, 10, 11, 12})
		So(e, ShouldEqual, nil)

		obj, e = store.Get("hi")
		So(len(obj), ShouldEqual, 0)
		obj, e = store.Get("bye")
		So(len(obj), ShouldEqual, 8)
	})

}
