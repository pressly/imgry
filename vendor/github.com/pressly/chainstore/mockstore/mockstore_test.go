package mockstore

import (
	"testing"
	"time"

	"github.com/pressly/chainstore"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

// TestMockStoreSuccess creates and test a mockstore that always succeeds.
func TestMockStoreSuccess(t *testing.T) {
	var store chainstore.Store
	var err error
	var obj []byte

	ctx := context.Background()

	store = chainstore.New(New(&Config{
		Capacity:    100,
		SuccessRate: 1.0, // always succeeds.
	}))

	assert := assert.New(t)

	err = store.Put(ctx, "notnil", []byte("something"))
	assert.Nil(err)

	err = store.Put(ctx, "hi", []byte{1, 2, 3})
	assert.Nil(err)

	obj, err = store.Get(ctx, "hi")
	assert.Nil(err)
	assert.Equal(obj, []byte{1, 2, 3})

	err = store.Put(ctx, "bye", []byte{5, 6, 7, 8, 9, 10, 11, 12})
	assert.Nil(err)

	err = store.Del(ctx, "hi")
	assert.Nil(err)

	obj, err = store.Get(ctx, "hi")
	assert.NotNil(err)
	assert.Equal(len(obj), 0)

	obj, err = store.Get(ctx, "bye")
	assert.Nil(err)
	assert.Equal(len(obj), 8)

	obj, err = store.Get(ctx, "notnil")
	assert.Nil(err)
	assert.Equal(len(obj), 9)
}

// TestMockStoreFail creates and test a mockstore that always fails.
func TestMockStoreFail(t *testing.T) {
	var store chainstore.Store
	var err error

	ctx := context.Background()

	store = chainstore.New(New(&Config{
		Capacity:    100,
		SuccessRate: 0.0, // always succeeds.
	}))

	assert := assert.New(t)

	err = store.Put(ctx, "notnil", []byte("something"))
	assert.NotNil(err)

	err = store.Put(ctx, "hi", []byte{1, 2, 3})
	assert.NotNil(err)

	_, err = store.Get(ctx, "hi")
	assert.NotNil(err)

	err = store.Put(ctx, "bye", []byte{5, 6, 7, 8, 9, 10, 11, 12})
	assert.NotNil(err)

	err = store.Del(ctx, "hi")
	assert.NotNil(err)

	_, err = store.Get(ctx, "hi")
	assert.NotNil(err)

	_, err = store.Get(ctx, "bye")
	assert.NotNil(err)

	_, err = store.Get(ctx, "notnil")
	assert.NotNil(err)
}

// TestMockStoreCancelWithTimeout creates and test a mockstore with that after
// a while gets cancelled.
func TestMockStoreCancelWithTimeout(t *testing.T) {
	var store chainstore.Store
	var err error

	assert := assert.New(t)

	store = chainstore.New(New(&Config{
		Capacity:    100,
		SuccessRate: 1.0,             // always succeeds.
		Delay:       time.Second * 1, // any operation takes 1s.
	}))

	ctx, _ := context.WithTimeout(context.Background(), time.Millisecond*500)

	// After 0.5s this all is going to fail, because the context timed out.
	err = store.Put(ctx, "notnil", []byte("something"))
	assert.NotNil(err)

	err = store.Put(ctx, "hi", []byte{1, 2, 3})
	assert.NotNil(err)

	_, err = store.Get(ctx, "hi")
	assert.NotNil(err)

	err = store.Put(ctx, "bye", []byte{5, 6, 7, 8, 9, 10, 11, 12})
	assert.NotNil(err)

	err = store.Del(ctx, "hi")
	assert.NotNil(err)

	_, err = store.Get(ctx, "hi")
	assert.NotNil(err)

	_, err = store.Get(ctx, "bye")
	assert.NotNil(err)

	_, err = store.Get(ctx, "notnil")
	assert.NotNil(err)
}

// TestMockStoreCancelWithFunc creates and test a mockstore that succeeds at
// first but then is cancelled.
func TestMockStoreCancelWithFunc(t *testing.T) {
	var store chainstore.Store
	var err error

	ctx, cancel := context.WithCancel(context.Background())

	store = chainstore.New(New(&Config{
		Capacity:    100,
		SuccessRate: 1.0,             // always succeeds.
		Delay:       time.Second * 1, // any operation takes 1s.
	}))

	go func() {
		time.Sleep(time.Millisecond * 1500)
		cancel()
	}()

	assert := assert.New(t)

	// This is going to succeed.
	err = store.Put(ctx, "notnil", []byte("something"))
	assert.Nil(err)

	// This will fail because after 1.5s the context will send a cancellation signal.
	err = store.Put(ctx, "hi", []byte{1, 2, 3})
	assert.NotNil(err)

	_, err = store.Get(ctx, "hi")
	assert.NotNil(err)

	err = store.Put(ctx, "bye", []byte{5, 6, 7, 8, 9, 10, 11, 12})
	assert.NotNil(err)

	err = store.Del(ctx, "hi")
	assert.NotNil(err)

	_, err = store.Get(ctx, "hi")
	assert.NotNil(err)

	_, err = store.Get(ctx, "bye")
	assert.NotNil(err)

	_, err = store.Get(ctx, "notnil")
	assert.NotNil(err)
}

// TestMockStoreCancelWithDefaultTimeout tests automatic operation cancellation
// and posterior recover.
func TestMockStoreCancelWithDefaultTimeout(t *testing.T) {
	var store chainstore.Store
	var err error

	ctx := context.Background()

	cfg := &Config{
		Capacity:    100,
		SuccessRate: 1.0,             // always succeeds.
		Delay:       time.Second * 1, // any operation takes 0.5s.
	}

	store = chainstore.New(New(cfg))

	assert := assert.New(t)

	// This is going to fail because the timeout is lower than the delay.
	cfg.Timeout = time.Millisecond * 500

	err = store.Put(ctx, "notnil", []byte("something"))
	assert.Equal(chainstore.ErrTimeout, err)

	// This is going to succeed because the timeout is greater than the delay.
	cfg.Timeout = time.Millisecond * 1500

	err = store.Put(ctx, "notnil", []byte("something"))
	assert.Nil(err)

	// This is going to fail because the timeout is lower than the delay.
	cfg.Timeout = time.Millisecond * 500

	_, err = store.Get(ctx, "notnil")
	assert.Equal(chainstore.ErrTimeout, err)

	// This is going to succeed because the timeout is greater than the delay.
	cfg.Timeout = time.Millisecond * 1500

	_, err = store.Get(ctx, "notnil")
	assert.Nil(err)
}
