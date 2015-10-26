package lrumgr

import (
	"container/list"
	"errors"
	"sync"

	"github.com/pressly/chainstore"
)

type LruManager struct {
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

// TODO: should lruManager support a chain of passed stores..?
func New(capacity int64, store chainstore.Store) *LruManager {
	return &LruManager{
		store:    store,
		capacity: capacity,
		cushion:  int64(float64(capacity) * 0.1),
		items:    make(map[string]*lruItem, 10000),
		list:     list.New(),
	}
}

func (m *LruManager) Open() (err error) {
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

func (m *LruManager) Close() (err error) {
	return m.store.Close()
}

func (m *LruManager) Put(key string, val []byte) (err error) {
	defer m.prune() // free up space

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
	return m.store.Put(key, val)
}

func (m *LruManager) Get(key string) (val []byte, err error) {
	val, err = m.store.Get(key)
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

func (m *LruManager) Del(key string) (err error) {
	m.Lock()
	if item, exists := m.items[key]; exists {
		m.evict(item)
	}
	m.Unlock()

	return m.store.Del(key)
}

func (m *LruManager) Capacity() int64 {
	m.Lock()
	defer m.Unlock()
	return m.capacity
}

func (m *LruManager) Cushion() int64 {
	return m.cushion
}

func (m *LruManager) NumItems() int {
	m.Lock()
	defer m.Unlock()
	return m.list.Len()
}

func (m *LruManager) addItem(key string, size int64) {
	item := &lruItem{key: key, size: size}
	item.listElement = m.list.PushFront(item)
	m.items[key] = item
	m.capacity -= size
}

func (m *LruManager) evict(item *lruItem) {
	m.list.Remove(item.listElement)
	delete(m.items, item.key)
	m.capacity += item.size
}

func (m *LruManager) prune() {
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

		m.Del(item.key)
	}

	// if m.capacity < 0 {
	// 	m.prune()
	// }
}
