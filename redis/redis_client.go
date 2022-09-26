package redis

import (
	"time"

	"github.com/go-redis/redis"
)

const (
	ErrNotFound = 404

	ErrNotFoundStr = "not found"
)

type RedisClientErr struct {
	code int
	msg  string
}

func (e *RedisClientErr) Code() int {
	return e.code
}

func (e *RedisClientErr) Error() string {
	return e.msg
}

func NewRedisClientErr(code int, msg string) *RedisClientErr {
	return &RedisClientErr{
		code: code,
		msg:  msg,
	}
}

func NewRedisNotFoundErr() *RedisClientErr {
	return NewRedisClientErr(ErrNotFound, ErrNotFoundStr)
}

type RedisStore interface {
	Ping() (err error)
	Set(key string, value interface{}) error
	SetWithExp(key string, value interface{}, expiration time.Duration) error
	HExists(key string, field string) error
	HGet(key string) (map[string]string, error)
	HSet(key string, m map[string]interface{}) error
	Delete(key string) error
	Get(key string) (string, error)
	Client() *redis.Client
	Close() error
}

type RedisClient struct {
	client *redis.Client
}

func NewRedisClient(addr, pass string, maxRetries int) RedisStore {
	opt := &redis.Options{
		Addr: addr,
	}
	if pass != "" {
		opt.Password = pass
	}
	if maxRetries > 0 && maxRetries < 5 {
		opt.MaxRetries = maxRetries
	}
	return &RedisClient{
		client: redis.NewClient(opt),
	}
}

func isErrNotFound(err error) bool {
	return err == redis.Nil
}

func (c *RedisClient) Close() error {
	return c.client.Close()
}

func (c *RedisClient) Ping() (err error) {
	_, err = c.client.Ping().Result()
	return
}

func (c *RedisClient) Set(key string, value interface{}) error {
	return c.client.Set(key, value, 0).Err()
}

// will return not found when dne
func (c *RedisClient) HExists(key string, field string) error {
	x, e := c.client.HExists(key, field).Result()
	if e != nil {
		return e
	}
	if !x {
		return NewRedisNotFoundErr()
	}
	return nil
}

func (c *RedisClient) HGet(key string) (map[string]string, error) {
	m, e := c.client.HGetAll(key).Result()
	if isErrNotFound(e) {
		return nil, NewRedisNotFoundErr()
	}
	return m, e
}

func (c *RedisClient) HSet(key string, m map[string]interface{}) error {
	return c.client.HMSet(key, m).Err()
}

func (c *RedisClient) SetWithExp(key string, value interface{}, expiration time.Duration) error {
	return c.client.Set(key, value, expiration).Err()
}

func (c *RedisClient) Get(key string) (v string, e error) {
	v, e = c.client.Get(key).Result()
	if isErrNotFound(e) {
		return "", NewRedisNotFoundErr()
	}
	return v, e
}

func (c *RedisClient) Delete(key string) error {
	err := c.client.Del(key).Err()
	if err == redis.Nil {
		return NewRedisNotFoundErr()
	}
	return err
}

func (c *RedisClient) Client() *redis.Client {
	return c.client
}
