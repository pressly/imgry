package logmgr

import (
	"fmt"
	"log"

	"github.com/pressly/chainstore"
	"golang.org/x/net/context"
)

type logManager struct {
	logger *log.Logger
	tag    string
}

// NOTE: this will chirp too often when put`ing back up the chain
// after a get. we may need to make somedistinction between stores and mangers

// New returns a logger.
func New(logger *log.Logger, tag string) chainstore.Store {
	if tag != "" {
		tag = fmt.Sprintf(" [%s]", tag)
	}
	return &logManager{logger, tag}
}

func (m *logManager) Open() error {
	return nil
}

func (m *logManager) Close() error {
	return nil
}

func (m *logManager) Put(ctx context.Context, key string, val []byte) (err error) {
	select {
	case <-ctx.Done():
		m.logger.Printf("chainstore%s: Put %s of %d bytes (cancelled)", m.tag, key, len(val))
		return ctx.Err()
	default:
		m.logger.Printf("chainstore%s: Put %s of %d bytes", m.tag, key, len(val))
		return
	}
}

func (m *logManager) Get(ctx context.Context, key string) (val []byte, err error) {
	select {
	case <-ctx.Done():
		m.logger.Printf("chainstore%s: Get %s (cancelled)", m.tag, key)
		return nil, ctx.Err()
	default:
		m.logger.Printf("chainstore%s: Get %s", m.tag, key)
		return
	}
}

func (m *logManager) Del(ctx context.Context, key string) (err error) {
	select {
	case <-ctx.Done():
		m.logger.Printf("chainstore%s: Del %s (cancelled)", m.tag, key)
		return ctx.Err()
	default:
		m.logger.Printf("chainstore%s: Del %s", m.tag, key)
		return
	}
}
