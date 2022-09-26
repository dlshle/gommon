package redis

import (
	"sync/atomic"

	"github.com/dlshle/gommon/utils"
)

type Jsonifiable interface {
	Json() string
}

type SingleEntityStoreV2[T Jsonifiable] interface {
	Get(id string) (T, error)
	Update(id string, value T) error
	Create(id string, value T) error
	Delete(id string) error
}

type redisEntityStore[T Jsonifiable] struct {
	store                 SingleEntityStoreV2[T]
	cache                 *RedisClient
	cacheOnCreate         bool
	skipErrOnCacheFailure bool
	writePolicy           uint8
	jsonDataUnmarshaller  func(data string) (T, error)
	hitCount              uint32
	missCount             uint32
}

func NewRedisEntityStore[T Jsonifiable](store SingleEntityStoreV2[T], cache *RedisClient, cacheOnCreate bool, skipErrOnCacheFailure bool, writePolicy uint8) SingleEntityStoreV2[T] {
	return &redisEntityStore[T]{
		store:                 store,
		cache:                 cache,
		cacheOnCreate:         cacheOnCreate,
		skipErrOnCacheFailure: skipErrOnCacheFailure,
		writePolicy:           writePolicy,
		jsonDataUnmarshaller: func(data string) (T, error) {
			return utils.UnmarshalJSONEntity[T]([]byte(data))
		},
		hitCount:  0,
		missCount: 0,
	}
}

func (s *redisEntityStore[T]) Get(id string) (entity T, err error) {
	jsonEntity, err := s.cache.Get(id)
	if err == nil {
		s.cacheHit(id)
	}
	if err != nil && err.Error() != ErrNotFoundStr {
		// conn error
		return
	}
	// hget does not return NotFoundErr but an empty map
	if err != nil && err.Error() == ErrNotFoundStr || jsonEntity == "" {
		s.cacheMiss(id)
		entity, err = s.store.Get(id)
		if err != nil {
			return
		}
		jsonified := entity.Json()
		err = s.cache.Set(id, jsonified)
		if s.skipErrOnCacheFailure {
			err = nil
		}
		return
	}
	return s.jsonDataUnmarshaller(jsonEntity)
}

func (s *redisEntityStore[T]) Update(id string, value T) error {
	j := value.Json()
	switch s.writePolicy {
	case CachePolicyWriteThrough:
		return s.writeThroughSet(id, value, j)
	default:
		return s.writeBackSet(id, value, j)
	}
}

func (s *redisEntityStore[T]) writeThroughSet(id string, entity T, j string) (err error) {
	return s.writeThroughAction(func() error { return s.store.Update(id, entity) }, func() error { return s.cache.Set(id, j) })
}

func (s *redisEntityStore[T]) writeBackSet(id string, entity T, j string) (err error) {
	return s.writeBackAction(func() error { return s.store.Update(id, entity) }, func() error { return s.cache.Set(id, j) })
}

func (s *redisEntityStore[T]) writeThroughAction(storeAction func() error, cacheAction func() error) error {
	if err := cacheAction(); err != nil {
		return err
	}
	return storeAction()
}

func (s *redisEntityStore[T]) writeBackAction(storeAction func() error, cacheAction func() error) error {
	if err := storeAction(); err != nil {
		return err
	}
	return cacheAction()
}

func (s *redisEntityStore[T]) Create(id string, value T) (err error) {
	if err = s.store.Create(id, value); err != nil {
		return
	}
	jsonified := value.Json()
	if err != nil {
		return err
	}
	if s.cacheOnCreate {
		return s.cache.Set(id, jsonified)
	}
	return nil
}

func (s *redisEntityStore[T]) Delete(id string) error {
	switch s.writePolicy {
	case CachePolicyWriteThrough:
		return s.writeThroughAction(func() error { return s.store.Delete(id) }, func() error { return s.cacheSafeDelete(id) })
	default:
		return s.writeBackAction(func() error { return s.store.Delete(id) }, func() error { return s.cacheSafeDelete(id) })
	}
}

func (s *redisEntityStore[T]) cacheSafeDelete(key string) error {
	err := s.Delete(key)
	if err != nil && err.Error() == ErrNotFoundStr {
		return nil
	}
	return err
}

func (s *redisEntityStore[T]) cacheHit(id string) {
	atomic.AddUint32(&s.hitCount, 1)
}

func (s *redisEntityStore[T]) cacheMiss(id string) {
	atomic.AddUint32(&s.missCount, 1)
}
