package s3store_test

import (
	"testing"

	"github.com/pressly/chainstore"
	. "github.com/smartystreets/goconvey/convey"
)

func TestS3Store(t *testing.T) {
	var store chainstore.Store
	var err error

	_ = store
	_ = err

	Convey("S3 Open", t, func() {
		// TODO
	})
}
