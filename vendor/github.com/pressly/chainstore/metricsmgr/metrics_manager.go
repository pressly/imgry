package metricsmgr

import (
	"fmt"
	"time"

	"github.com/pressly/chainstore"
	"github.com/rcrowley/go-metrics"
)

type metricsManager struct {
	namespace string
	registry  metrics.Registry
	chain     chainstore.Store
}

func New(namespace string, registry metrics.Registry, stores ...chainstore.Store) *metricsManager {
	return &metricsManager{
		namespace: namespace,
		registry:  registry,
		chain:     chainstore.New(stores...),
	}
}

func (m *metricsManager) Open() (err error) {
	_, err = m.measure("Open", func() ([]byte, error) {
		err := m.chain.Open()
		return nil, err
	})
	return
}

func (m *metricsManager) Close() (err error) {
	_, err = m.measure("Close", func() ([]byte, error) {
		err := m.chain.Close()
		return nil, err
	})
	return
}

func (m *metricsManager) Put(key string, val []byte) (err error) {
	_, err = m.measure("Put", func() ([]byte, error) {
		err := m.chain.Put(key, val)
		return nil, err
	})
	return
}

func (m *metricsManager) Get(key string) (val []byte, err error) {
	val, err = m.measure("Get", func() ([]byte, error) {
		val, err := m.chain.Get(key)
		return val, err
	})
	return
}

func (m *metricsManager) Del(key string) (err error) {
	_, err = m.measure("Del", func() ([]byte, error) {
		err := m.chain.Del(key)
		return nil, err
	})
	return
}

func (m *metricsManager) measure(method string, fn func() ([]byte, error)) ([]byte, error) {
	ns := fmt.Sprintf("%s.%s", m.namespace, method)
	metric := metrics.GetOrRegisterTimer(ns, m.registry)
	t := time.Now()
	val, err := fn()
	metric.UpdateSince(t)
	return val, err
}
