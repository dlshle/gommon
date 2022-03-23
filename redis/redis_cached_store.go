package redis

import (
	"github.com/dlshle/gommon/logger"
)

const (
	CachePolicyWriteThrough = 1
	CachePolicyWriteBack    = 2

	CacheMark = "RCS-mark"
)

type ISingleEntityStore interface {
	Get(id string) (interface{}, error)
	Update(id string, value interface{}) error
	Create(id string, value interface{}) error
	Delete(id string) error
	ToHashMap(interface{}) (map[string]interface{}, error)
	ToEntityType(map[string]string) (interface{}, error)
}

type RedisCachedStore struct {
	store                 ISingleEntityStore
	cache                 *RedisClient
	cacheOnCreate         bool
	skipErrOnCacheFailure bool
	writePolicy           uint8
	logger                *logger.SimpleLogger
}

func NewRedisCachedStore(logger *logger.SimpleLogger, store ISingleEntityStore, cache *RedisClient, cacheOnCreate bool, skipErrOnCacheFailure bool, writePolicy uint8) *RedisCachedStore {
	if writePolicy > CachePolicyWriteBack {
		writePolicy = CachePolicyWriteBack
	}
	return &RedisCachedStore{
		store:                 store,
		cache:                 cache,
		cacheOnCreate:         cacheOnCreate,
		skipErrOnCacheFailure: skipErrOnCacheFailure,
		writePolicy:           writePolicy,
		logger:                logger,
	}
}

func (s *RedisCachedStore) Ping() error {
	return s.cache.Ping()
}

func (s *RedisCachedStore) ToHashMap(entity interface{}) (map[string]interface{}, error) {
	m, e := s.store.ToHashMap(entity)
	if e != nil {
		return nil, e
	}
	m[CacheMark] = true
	return m, e
}

func (s *RedisCachedStore) checkAndGet(id string) (map[string]string, error) {
	err := s.cache.HExists(id, CacheMark)
	if err != nil {
		return nil, err
	}
	return s.cache.HGet(id)
}

func (s *RedisCachedStore) Get(id string) (entity interface{}, err error) {
	var m map[string]string
	m, err = s.checkAndGet(id)
	if err == nil {
		s.logger.Printf("Fetch %s hit", id)
	}
	if err != nil && err.Error() != ErrNotFoundStr {
		// conn error
		return
	}
	if err != nil && err.Error() == ErrNotFoundStr {
		s.logger.Printf("Fetch %s miss", id)
		entity, err = s.store.Get(id)
		if err != nil {
			return
		}
		hm, terr := s.ToHashMap(entity)
		if terr != nil {
			err = terr
			return
		}
		err = s.cache.HSet(id, hm)
		if s.skipErrOnCacheFailure {
			err = nil
		}
		return
	}
	return s.store.ToEntityType(m)
}

func (s *RedisCachedStore) Update(id string, value interface{}) error {
	m, err := s.ToHashMap(value)
	if err != nil {
		return err
	}
	switch s.writePolicy {
	case CachePolicyWriteThrough:
		return s.writeThroughSet(id, value, m)
	default:
		return s.writeBackSet(id, value, m)
	}
}

func (s *RedisCachedStore) writeThroughSet(id string, entity interface{}, m map[string]interface{}) (err error) {
	return s.writeThroughAction(func() error { return s.store.Update(id, entity) }, func() error { return s.cache.HSet(id, m) })
}

func (s *RedisCachedStore) writeBackSet(id string, entity interface{}, m map[string]interface{}) (err error) {
	return s.writeBackAction(func() error { return s.store.Update(id, entity) }, func() error { return s.cache.HSet(id, m) })
}

func (s *RedisCachedStore) writeThroughAction(storeAction func() error, cacheAction func() error) error {
	if err := cacheAction(); err != nil {
		return err
	}
	return storeAction()
}

func (s *RedisCachedStore) writeBackAction(storeAction func() error, cacheAction func() error) error {
	if err := storeAction(); err != nil {
		return err
	}
	return cacheAction()
}

func (s *RedisCachedStore) Create(id string, value interface{}) (err error) {
	if err = s.store.Create(id, value); err != nil {
		return
	}
	m, err := s.ToHashMap(value)
	if err != nil {
		return err
	}
	if s.cacheOnCreate {
		return s.cache.HSet(id, m)
	}
	return nil
}

func (s *RedisCachedStore) cacheSafeDelete(key string) error {
	err := s.Delete(key)
	if err != nil && err.Error() == ErrNotFoundStr {
		return nil
	}
	return err
}

func (s *RedisCachedStore) Delete(id string) error {
	switch s.writePolicy {
	case CachePolicyWriteThrough:
		return s.writeThroughAction(func() error { return s.store.Delete(id) }, func() error { return s.cacheSafeDelete(id) })
	default:
		return s.writeBackAction(func() error { return s.store.Delete(id) }, func() error { return s.cacheSafeDelete(id) })
	}
}
