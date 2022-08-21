package store

type KVStore[K, V comparable] interface {
	Get(key K) (res V, err error)
	Has(key K) (bool, error)
	Put(key K, value V) (success bool, err error)
	Update(key K, value V) (success bool, err error)
	Delete(key K) (bool, error)
	Query(filter func(key K, record V) bool) (res []V, err error)
	Iterate(itr func(key K, record V)) error
	BulkGet(keys []K) (res []V, err error)
	BulkPut(bulk map[K]V) (success bool, err error)
	Close() error
	Drop() error
}

type AutoIncrKVStore[K, V comparable] interface {
	KVStore[K, V]
	Create(V) (K, V, error)
}
