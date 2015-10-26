package logmgr

import (
	"fmt"
	"log"
)

type logManager struct {
	logger *log.Logger
	tag    string
}

// NOTE: this will chirp too often when put`ing back up the chain
// after a get. we may need to make somedistinction between stores and mangers

func New(logger *log.Logger, tag string) *logManager {
	if tag != "" {
		tag = fmt.Sprintf(" [%s]", tag)
	}
	return &logManager{logger, tag}
}

func (m *logManager) Open() (err error)  { return }
func (m *logManager) Close() (err error) { return }

func (m *logManager) Put(key string, val []byte) (err error) {
	m.logger.Printf("chainstore%s: Put %s of %d bytes", m.tag, key, len(val))
	return
}

func (m *logManager) Get(key string) (val []byte, err error) {
	m.logger.Printf("chainstore%s: Get %s", m.tag, key)
	return
}

func (m *logManager) Del(key string) (err error) {
	m.logger.Printf("chainstore%s: Del %s", m.tag, key)
	return
}
