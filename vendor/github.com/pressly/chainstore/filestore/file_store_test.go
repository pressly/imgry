package filestore

import (
	"io/ioutil"
	"testing"

	"github.com/pressly/chainstore"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func tempDir() string {
	path, _ := ioutil.TempDir("", "chainstore-")
	return path
}

func TestFileStore(t *testing.T) {
	var store chainstore.Store
	var err error

	ctx := context.Background()

	store = chainstore.New(New(tempDir(), 0755))

	assert := assert.New(t)

	err = store.Open()
	assert.Nil(err)

	// Put/Get/Del basic data
	err = store.Put(ctx, "test.txt", []byte{1, 2, 3, 4})
	assert.Nil(err)

	data, err := store.Get(ctx, "test.txt")
	assert.Nil(err)
	assert.Equal(data, []byte{1, 2, 3, 4})

	// Auto-creating directories on put
	err = store.Put(ctx, "hello/there/everyone.txt", []byte{1, 2, 3, 4})
	assert.Nil(err)
}
