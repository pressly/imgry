package chainstore_test

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pressly/chainstore"
	"github.com/pressly/chainstore/boltstore"
	"github.com/pressly/chainstore/filestore"
	"github.com/pressly/chainstore/logmgr"
	"github.com/pressly/chainstore/lrumgr"
	"github.com/pressly/chainstore/memstore"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func tempDir() string {
	path, _ := ioutil.TempDir("", "chainstore-")
	return path
}

func TestBasicChain(t *testing.T) {
	var ms, fs, chain chainstore.Store
	var err error

	ctx := context.Background()

	logger := log.New(os.Stdout, "", log.LstdFlags)

	storeDir := tempDir()
	err = nil

	ms = memstore.New(100)
	fs = filestore.New(storeDir+"/filestore", 0755)

	chain = chainstore.New(
		logmgr.New(logger, ""),
		ms,
		fs,
	)

	assert := assert.New(t)

	err = chain.Open()
	assert.Nil(err)

	v := []byte("value")
	err = chain.Put(ctx, "k", v)
	assert.Nil(err)

	val, err := chain.Get(ctx, "k")
	assert.Nil(err)
	assert.Equal(val, v)

	val, err = ms.Get(ctx, "k")
	assert.Nil(err)
	assert.Equal(val, v)

	val, err = fs.Get(ctx, "k")
	assert.Nil(err)
	assert.Equal(val, v)

	err = chain.Del(ctx, "k")
	assert.Nil(err)

	val, err = fs.Get(ctx, "k")
	assert.Nil(err)
	assert.Equal(len(val), 0)

	val, err = chain.Get(ctx, "woo!@#")
	assert.NotNil(err)
}

func TestAsyncChain(t *testing.T) {
	var ms, fs, bs, chain chainstore.Store
	var err error

	logger := log.New(os.Stdout, "", log.LstdFlags)
	storeDir := tempDir()

	var errored atomic.Value

	ms = memstore.New(100)
	fs = filestore.New(storeDir+"/filestore", 0755)
	bs = boltstore.New(storeDir+"/boltstore/bolt.db", "test")

	chain = chainstore.New(
		logmgr.New(logger, ""),
		ms,
		chainstore.Async(
			func(err error) {
				log.Println("async error:", err)
				errored.Store(true)
			},
			logmgr.New(logger, "async"),
			&testStore{},
			chainstore.New(
				fs,
				lrumgr.New(100, bs),
			),
		),
	)

	ctx := context.Background()
	assert := assert.New(t)

	err = chain.Open()
	assert.Nil(err)

	v := []byte("value")
	err = chain.Put(ctx, "k", v)
	assert.Nil(err)

	val, err := chain.Get(ctx, "k")
	assert.Nil(err)
	assert.Equal(val, v)

	val, err = ms.Get(ctx, "k")
	assert.Nil(err)
	assert.Equal(val, v)

	time.Sleep(time.Second * 1) // wait for async operation..

	val, err = fs.Get(ctx, "k")
	assert.Nil(err)
	assert.Equal(val, v)

	val, err = bs.Get(ctx, "k")
	assert.Nil(err)
	assert.Equal(val, v)

	//--

	// Lets make an error in async store
	assert.Nil(errored.Load())

	err = chain.Put(ctx, "bad", []byte("v"))
	assert.Nil(err) // no error because sync store took it fine

	time.Sleep(time.Second * 1) // wait for async operation..
	assert.NotEmpty(errored.Load())
}

type testStore struct{}

func (s *testStore) Open() (err error)  { return }
func (s *testStore) Close() (err error) { return }

func (s *testStore) Put(ctx context.Context, key string, val []byte) (err error) {
	if key == "bad" {
		return errors.New("testStore: err")
	} else {
		return nil
	}
}

func (s *testStore) Get(ctx context.Context, key string) (data []byte, err error) {
	if key == "bad" {
		return nil, errors.New("testStore: err")
	} else {
		return []byte("ok"), nil
	}
}

func (s *testStore) Del(ctx context.Context, key string) (err error) {
	if key == "bad" {
		return errors.New("testStore: err")
	} else {
		return nil
	}
}
