package server

import (
	"errors"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/goware/go-metrics"
)

var (
	ErrDBGetKey = errors.New("DB: Unable to get the key")
)

type DB struct {
	pool *redis.Pool
}

func NewDB(address string) (*DB, error) {
	pool := &redis.Pool{
		MaxIdle:     64,
		MaxActive:   64,
		IdleTimeout: 300 * time.Second,
		Wait:        true,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", address)
			if err != nil {
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
	return &DB{pool: pool}, nil
}

func (db *DB) Close() {
	db.pool.Close()
}

func (db *DB) Ping() error {
	conn := db.conn()
	defer conn.Close()
	if _, err := conn.Do("PING"); err != nil {
		return err
	}
	return nil
}

func (db *DB) Get(key string) (val []byte, err error) {
	defer metrics.MeasureSince([]string{"fn.redis.Get"}, time.Now())

	conn := db.conn()
	defer conn.Close()
	reply, err := conn.Do("GET", key)
	if err != nil {
		return nil, err
	}
	val, ok := reply.([]byte)
	if !ok {
		return nil, ErrDBGetKey
	}
	return
}

func (db *DB) Set(key string, obj []byte, expireIn ...time.Duration) (err error) {
	defer metrics.MeasureSince([]string{"fn.redis.Set"}, time.Now())

	conn := db.conn()
	defer conn.Close()

	var ex int64
	if len(expireIn) > 0 {
		ex = int64(expireIn[0].Seconds())
	}

	if ex > 0 {
		_, err = conn.Do("SETEX", key, ex, obj)
	} else {
		_, err = conn.Do("SET", key, obj)
	}
	return
}

func (db *DB) Del(key string) (err error) {
	conn := db.conn()
	defer conn.Close()
	_, err = conn.Do("DEL", key)
	return
}

func (db *DB) Exists(key string) (bool, error) {
	conn := db.conn()
	defer conn.Close()
	reply, err := conn.Do("EXISTS", key)
	if n, ok := reply.(int64); ok {
		if n == 1 {
			return true, nil
		} else {
			return false, nil
		}
	}
	return false, err
}

func (db *DB) HGet(key string, dest interface{}) error {
	defer metrics.MeasureSince([]string{"fn.redis.HGet"}, time.Now())

	conn := db.conn()
	defer conn.Close()
	reply, err := redis.Values(conn.Do("HGETALL", key))
	if err != nil {
		return err
	}
	return redis.ScanStruct(reply, dest)
}

func (db *DB) HSet(key string, src interface{}) error {
	defer metrics.MeasureSince([]string{"fn.redis.HSet"}, time.Now())

	conn := db.conn()
	defer conn.Close()
	_, err := conn.Do("HMSET", redis.Args{}.Add(key).AddFlat(src)...)
	return err
}

func (db *DB) conn() redis.Conn {
	return db.pool.Get()
}
