package auth

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/garyburd/redigo/redis"
)

type RedisAuth struct {
	Address  string
	TTL      int
	Limit    int
	Username string
	Password string

	pool *redis.Pool
}

func (ra *RedisAuth) Connect() {
	ra.pool = &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			con, err := redis.Dial("tcp", ra.Address)
			if err != nil {
				return nil, err
			}
			return con, err
		},
		TestOnBorrow: func(con redis.Conn, t time.Time) error {
			_, err := con.Do("PING")
			return err
		},
	}
}

func (ra *RedisAuth) Handler() func(http.Handler) http.Handler {
	return Wrap(ra.handler, "Restricted")
}

func (ra *RedisAuth) handler(r *http.Request, opts []string) bool {
	authString := fmt.Sprintf("%s:%s", ra.Username, ra.Password)
	auth := r.Header.Get("Authorization")
	prefix := "Basic "
	if !strings.HasPrefix(auth, prefix) {
		return false
	}
	givenSecret := auth[len(prefix):]
	decodedSecret, err := base64.StdEncoding.DecodeString(givenSecret)
	if err != nil {
		return false
	}
	user := strings.Split(string(decodedSecret), ":")[0]

	blocked, err := ra.checkUser(user)
	if err != nil {
		return false
	}
	if blocked {
		log.Printf("%s is blocked", user)
		return false
	}

	p, err := decodePlainAuth(givenSecret)
	if err != nil || p != authString {
		log.Println("Failed to authorize", user)
		ra.failUser(user)
		return false
	}
	log.Println("Authorized", user)
	return true
}

func (ra *RedisAuth) checkUser(user string) (bool, error) {
	c := ra.pool.Get()
	defer c.Close()

	result, err := redis.Int(c.Do("GET", user))
	if err != nil {
		_, e := c.Do("SET", user, 0)
		if e != nil {
			return true, e
		}
		return true, nil
	}

	if result > ra.Limit {
		return true, nil
	}
	return false, nil
}

func (ra *RedisAuth) failUser(user string) error {
	c := ra.pool.Get()
	defer c.Close()

	c.Send("MULTI")
	c.Send("INCR", user)
	c.Send("EXPIRE", user, ra.TTL)
	_, err := c.Do("EXEC")
	return err
}
