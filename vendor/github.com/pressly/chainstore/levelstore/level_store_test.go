package levelstore

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

func TestLevelStore(t *testing.T) {
	var store chainstore.Store
	var err error

	ctx := context.Background()

	assert := assert.New(t)

	store = chainstore.New(New(tempDir()))
	err = store.Open()
	assert.Nil(err)
	defer store.Close()

	// Put a bunch of objects
	e1 := store.Put(ctx, "hi", []byte{1, 2, 3})
	e2 := store.Put(ctx, "bye", []byte{4, 5, 6})
	assert.Nil(e1)
	assert.Nil(e2)

	// Get those objects
	v1, _ := store.Get(ctx, "hi")
	v2, _ := store.Get(ctx, "bye")
	assert.Equal(v1, []byte{1, 2, 3})
	assert.Equal(v2, []byte{4, 5, 6})

	// Delete those objects
	e1 = store.Del(ctx, "hi")
	e2 = store.Del(ctx, "bye")
	assert.Equal(e1, nil)
	assert.Equal(e2, nil)

}
