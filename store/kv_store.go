package store

import badger "github.com/dgraph-io/badger/v3"

type KVStore[K, V comparable] interface {
	Get(key K) (res V, err error)
	GetWithTxn(tx *badger.Txn, key K) (res V, err error)
	Has(key K) (bool, error)
	Put(key K, value V) (success bool, err error)
	PutWithTxn(tx *badger.Txn, key K, value V) (err error)
	Update(key K, value V) (success bool, err error)
	UpdateWithTxn(tx *badger.Txn, key K, value V) (err error)
	Delete(key K) (bool, error)
	DeleteWithTx(tx *badger.Txn, key K) error
	Query(filter func(key K, record V) bool) (res []V, err error)
	Iterate(itr func(key K, record V)) error
	BulkGet(keys []K) (res []V, err error)
	BulkPut(bulk map[K]V) (success bool, err error)
	WithTx(cb func(*badger.Txn) error) error
	Close() error
	Drop() error
}

type AutoIncrKVStore[K, V comparable] interface {
	KVStore[K, V]
	Create(V) (K, V, error)
}
