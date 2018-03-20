package lrumgr

import (
	"testing"

	"github.com/pressly/chainstore"
	"github.com/pressly/chainstore/filestore"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
	"io/ioutil"
)

func tempDir() string {
	path, _ := ioutil.TempDir("", "chainstore-")
	return path
}

func TestLRUManager(t *testing.T) {
	var err error
	var store chainstore.Store
	var lru *lruManager
	var capacity int64 = 20

	ctx := context.Background()

	store = filestore.New(tempDir(), 0755)

	lru = newLruManager(capacity, store)

	assert := assert.New(t)

	// based on 10% cushion
	lru.Put(ctx, "peter", []byte{1, 2, 3})
	lru.Put(ctx, "jeff", []byte{4})
	lru.Put(ctx, "julia", []byte{5, 6, 7, 8, 9, 10})
	lru.Put(ctx, "janet", []byte{11, 12, 13})
	lru.Put(ctx, "ted", []byte{14, 15, 16, 17, 18})

	remaining := capacity - 18
	assert.Equal(lru.Capacity(), remaining)

	remaining = remaining + 4
	err = lru.Put(ctx, "agnes", []byte{20, 21, 22, 23, 24, 25})
	assert.Equal(lru.Capacity(), remaining)
	assert.Nil(err)

	var b []byte

	// has been evicted..
	b, err = lru.Get(ctx, "peter")
	assert.Nil(err)
	assert.Equal(len(b), 0)

	// exists
	b, err = lru.Get(ctx, "janet")
	assert.Nil(err)
	assert.Equal(b, []byte{11, 12, 13})
}
