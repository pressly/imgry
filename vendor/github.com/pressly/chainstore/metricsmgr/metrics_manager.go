package metricsmgr

import (
	"fmt"
	"time"

	"github.com/armon/go-metrics"
	"github.com/pressly/chainstore"
	"golang.org/x/net/context"
)

type metricsManager struct {
	namespace string
	chain     chainstore.Store
}

// New returns a metrics store.
func New(namespace string, stores ...chainstore.Store) chainstore.Store {
	return &metricsManager{
		namespace: namespace,
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

func (m *metricsManager) Put(ctx context.Context, key string, val []byte) (err error) {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		_, err = m.measure("Put", func() ([]byte, error) {
			err := m.chain.Put(ctx, key, val)
			return nil, err
		})
		return
	}
}

func (m *metricsManager) Get(ctx context.Context, key string) (val []byte, err error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		val, err = m.measure("Get", func() ([]byte, error) {
			val, err := m.chain.Get(ctx, key)
			return val, err
		})
		return
	}
}

func (m *metricsManager) Del(ctx context.Context, key string) (err error) {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		_, err = m.measure("Del", func() ([]byte, error) {
			err := m.chain.Del(ctx, key)
			return nil, err
		})
		return
	}
}

func (m *metricsManager) measure(method string, fn func() ([]byte, error)) ([]byte, error) {
	name := fmt.Sprintf("%s.%s", m.namespace, method)
	defer metrics.MeasureSince([]string{name}, time.Now())
	return fn()
}
