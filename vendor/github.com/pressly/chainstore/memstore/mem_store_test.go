package memstore

import (
	"testing"

	"github.com/pressly/chainstore"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestMemCacheStore(t *testing.T) {
	var store chainstore.Store
	var err error
	var obj []byte

	ctx := context.Background()

	store = chainstore.New(New(10))

	assert := assert.New(t)

	err = store.Put(ctx, "hi", []byte{1, 2, 3})
	assert.Nil(err)

	obj, err = store.Get(ctx, "hi")
	assert.Nil(err)
	assert.Equal(obj, []byte{1, 2, 3})

	err = store.Put(ctx, "bye", []byte{5, 6, 7, 8, 9, 10, 11, 12})
	assert.Nil(err)

	obj, err = store.Get(ctx, "hi")
	assert.Equal(len(obj), 0)

	obj, err = store.Get(ctx, "bye")
	assert.Equal(len(obj), 8)
}
