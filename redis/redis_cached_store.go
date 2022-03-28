package redis

import (
	"github.com/dlshle/gommon/logger"
)

const (
	CachePolicyWriteThrough = 1
	CachePolicyWriteBack    = 2

	CacheMark = "RCS-mark"
)

type SingleEntityStore interface {
	Get(id string) (interface{}, error)
	Update(id string, value interface{}) error
	Create(id string, value interface{}) error
	Delete(id string) error
	ToHashMap(interface{}) (map[string]interface{}, error)
	ToEntityType(map[string]string) (interface{}, error)
}

type CachedStore struct {
	store                 SingleEntityStore
	cache                 *RedisClient
	cacheOnCreate         bool
	skipErrOnCacheFailure bool
	writePolicy           uint8
	logger                *logger.SimpleLogger
}

func NewRedisCachedStore(logger *logger.SimpleLogger, store SingleEntityStore, cache *RedisClient, cacheOnCreate bool, skipErrOnCacheFailure bool, writePolicy uint8) *CachedStore {
	if writePolicy > CachePolicyWriteBack {
		writePolicy = CachePolicyWriteBack
	}
	return &CachedStore{
		store:                 store,
		cache:                 cache,
		cacheOnCreate:         cacheOnCreate,
		skipErrOnCacheFailure: skipErrOnCacheFailure,
		writePolicy:           writePolicy,
		logger:                logger,
	}
}

func (s *CachedStore) Ping() error {
	return s.cache.Ping()
}

func (s *CachedStore) ToHashMap(entity interface{}) (map[string]interface{}, error) {
	m, e := s.store.ToHashMap(entity)
	if e != nil {
		return nil, e
	}
	m[CacheMark] = true
	return m, e
}

func (s *CachedStore) checkAndGet(id string) (map[string]string, error) {
	err := s.cache.HExists(id, CacheMark)
	if err != nil {
		return nil, err
	}
	return s.cache.HGet(id)
}

func (s *CachedStore) Get(id string) (entity interface{}, err error) {
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

func (s *CachedStore) Update(id string, value interface{}) error {
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

func (s *CachedStore) writeThroughSet(id string, entity interface{}, m map[string]interface{}) (err error) {
	return s.writeThroughAction(func() error { return s.store.Update(id, entity) }, func() error { return s.cache.HSet(id, m) })
}

func (s *CachedStore) writeBackSet(id string, entity interface{}, m map[string]interface{}) (err error) {
	return s.writeBackAction(func() error { return s.store.Update(id, entity) }, func() error { return s.cache.HSet(id, m) })
}

func (s *CachedStore) writeThroughAction(storeAction func() error, cacheAction func() error) error {
	if err := cacheAction(); err != nil {
		return err
	}
	return storeAction()
}

func (s *CachedStore) writeBackAction(storeAction func() error, cacheAction func() error) error {
	if err := storeAction(); err != nil {
		return err
	}
	return cacheAction()
}

func (s *CachedStore) Create(id string, value interface{}) (err error) {
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

func (s *CachedStore) cacheSafeDelete(key string) error {
	err := s.Delete(key)
	if err != nil && err.Error() == ErrNotFoundStr {
		return nil
	}
	return err
}

func (s *CachedStore) Delete(id string) error {
	switch s.writePolicy {
	case CachePolicyWriteThrough:
		return s.writeThroughAction(func() error { return s.store.Delete(id) }, func() error { return s.cacheSafeDelete(id) })
	default:
		return s.writeBackAction(func() error { return s.store.Delete(id) }, func() error { return s.cacheSafeDelete(id) })
	}
}
