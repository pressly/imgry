package lrumgr

import (
	"container/list"
	"errors"
	"sync"

	"github.com/pressly/chainstore"
	"golang.org/x/net/context"
)

var _ = chainstore.Store(&lruManager{})

type lruManager struct {
	sync.Mutex

	store    chainstore.Store
	capacity int64 // in bytes
	cushion  int64 // 10% of bytes of the capacity, to free up this much if it hits

	items map[string]*lruItem
	list  *list.List
}

type lruItem struct {
	key         string
	size        int64
	listElement *list.Element
}

func newLruManager(capacity int64, store chainstore.Store) *lruManager {
	return &lruManager{
		store:    store,
		capacity: capacity,
		cushion:  int64(float64(capacity) * 0.1),
		items:    make(map[string]*lruItem, 10000),
		list:     list.New(),
	}
}

// New creates and returns a LRU backed store.
func New(capacity int64, store chainstore.Store) chainstore.Store {
	return newLruManager(capacity, store)
}

func (m *lruManager) Open() (err error) {

	if m.capacity < 10 {
		return errors.New("Invalid capacity, must be >= 10 bytes")
	}

	err = m.store.Open()

	// TODO: the items list will be empty after restarting a server
	// with an existing db. We should ask the store for a list of
	// keys and their size to seed this list. Keys are easy,
	// but having a generic way to get the size of each object quickly
	// from each kind of store is challenging / over-kill (ie. s3).
	// we could persist the LRU list of keys/objects somewhere..
	// perhaps using a bolt bucket.
	return
}

func (m *lruManager) Close() (err error) {
	return m.store.Close()
}

func (m *lruManager) Put(ctx context.Context, key string, val []byte) (err error) {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		defer m.prune(ctx) // free up space
		valSize := int64(len(val))

		m.Lock()
		if item, exists := m.items[key]; exists {
			m.list.MoveToFront(item.listElement)
			m.capacity += (item.size - valSize)
			item.size = valSize
			// m.promote(item)
		} else {
			m.addItem(key, valSize)
		}
		m.Unlock()

		// TODO: what if the value is larger then even the initial capacity?
		// ..error..
		return m.store.Put(ctx, key, val)
	}
}

func (m *lruManager) Get(ctx context.Context, key string) (val []byte, err error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		val, err = m.store.Get(ctx, key)
		valSize := len(val)

		m.Lock()
		if item, exists := m.items[key]; exists {
			// m.promote(item)
			m.list.MoveToFront(item.listElement)
		} else if valSize > 0 {
			m.addItem(key, int64(valSize))
		}
		m.Unlock()
		return
	}
}

func (m *lruManager) Del(ctx context.Context, key string) (err error) {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		m.Lock()

		if item, exists := m.items[key]; exists {
			m.evict(item)
		}
		m.Unlock()

		return m.store.Del(ctx, key)
	}
}

func (m *lruManager) Capacity() int64 {
	m.Lock()
	defer m.Unlock()
	return m.capacity
}

func (m *lruManager) Cushion() int64 {
	return m.cushion
}

func (m *lruManager) NumItems() int {
	m.Lock()
	defer m.Unlock()
	return m.list.Len()
}

func (m *lruManager) addItem(key string, size int64) {
	item := &lruItem{key: key, size: size}
	item.listElement = m.list.PushFront(item)
	m.items[key] = item
	m.capacity -= size
}

func (m *lruManager) evict(item *lruItem) {
	m.list.Remove(item.listElement)
	delete(m.items, item.key)
	m.capacity += item.size
}

func (m *lruManager) prune(ctx context.Context) {
	if m.capacity > 0 {
		return
	}

	for m.capacity < m.cushion {
		m.Lock()
		tail := m.list.Back()
		if tail == nil {
			return
		}
		item := tail.Value.(*lruItem)
		m.Unlock()

		m.Del(ctx, item.key)
	}
}
