package store

import (
	"encoding/binary"
	"time"

	"github.com/dlshle/gommon/errors"
	"github.com/dlshle/gommon/utils"

	badger "github.com/dgraph-io/badger/v3"
)

const keyIDRange uint64 = 999999999

type BadgerStoreSerializeHandler[K, V comparable] interface {
	KeySerializer(K) ([]byte, error)
	KeyDeserializer([]byte) (K, error)
	ValueSerializer(V) ([]byte, error)
	ValueDeserializer([]byte) (V, error)
}

type badgerStore[K, V comparable] struct {
	db  *badger.DB
	seq *badger.Sequence
	BadgerStoreSerializeHandler[K, V]
}

func newBadgerStore[K, V comparable](dbFilePath string, serializeHandler BadgerStoreSerializeHandler[K, V], useAutoIncrKey bool) (badgerStore[K, V], error) {
	var seq *badger.Sequence
	db, err := badger.Open(badger.DefaultOptions("./data/" + dbFilePath))
	if err != nil {
		return badgerStore[K, V]{}, err
	}
	if useAutoIncrKey {
		seq, err = db.GetSequence([]byte(dbFilePath), keyIDRange)
		if err != nil {
			return badgerStore[K, V]{}, err
		}
	}
	store := badgerStore[K, V]{
		db:                          db,
		seq:                         seq,
		BadgerStoreSerializeHandler: serializeHandler,
	}
	store.doGC()
	go store.garbageCollectionRoutine()
	return store, nil
}

func NewBadgerStore[K, V comparable](dbFilePath string, serializeHandler BadgerStoreSerializeHandler[K, V]) (KVStore[K, V], error) {
	return newBadgerStore(dbFilePath, serializeHandler, false)
}

func NewAutoIncrBadgerStore[K, V comparable](dbFilePath string, serializeHandler BadgerStoreSerializeHandler[K, V]) (AutoIncrKVStore[K, V], error) {
	return newBadgerStore(dbFilePath, serializeHandler, true)
}

func (s badgerStore[K, V]) garbageCollectionRoutine() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.doGC()
	}
}

func (s badgerStore[K, V]) doGC() {
again:
	// since gc removes only 1 file at a time, best
	// practice is to keep running till encountering an error
	err := s.db.RunValueLogGC(0.7)
	if err == nil {
		goto again
	}
}

func (s badgerStore[K, V]) withRead(cb func(tx *badger.Txn) error) error {
	return s.db.View(cb)
}

func (s badgerStore[K, V]) withWrite(cb func(tx *badger.Txn) error) error {
	return s.db.Update(cb)
}

func (s badgerStore[K, V]) Get(key K) (res V, err error) {
	err = s.withRead(func(tx *badger.Txn) error {
		res, err = s.GetWithTxn(tx, key)
		return err
	})
	return
}

func (s badgerStore[K, V]) GetWithTxn(tx *badger.Txn, key K) (res V, err error) {
	var serializedKey []byte
	serializedKey, err = s.KeySerializer(key)
	if err != nil {
		return
	}
	res, err = s.getValueBySerializedKey(tx, serializedKey)
	return
}

func (s badgerStore[K, V]) getValueBySerializedKey(tx *badger.Txn, key []byte) (value V, err error) {
	var item *badger.Item
	var zeroVal V
	err = utils.ProcessWithErrors(func() error {
		item, err = tx.Get(key)
		return err
	}, func() error {
		if item == nil {
			value = zeroVal
			return nil
		}
		return item.Value(func(val []byte) error {
			if item == nil {
				return nil
			}
			value, err = s.ValueDeserializer(val)
			return err
		})
	})
	return
}

func (s badgerStore[K, V]) Has(key K) (bool, error) {
	var zeroVal V
	val, err := s.Get(key)
	return val == zeroVal, err
}

func (s badgerStore[K, V]) Create(value V) (k K, v V, err error) {
	var (
		nextKey uint64
	)
	err = utils.ProcessWithErrors(func() error {
		if s.seq == nil {
			return errors.Error("create can not be applied in non-auto-incr-id store")
		}
		return nil
	}, func() error {
		nextKey, err = s.seq.Next()
		return err
	}, func() error {
		k, err = s.KeyDeserializer(uint64ToBytes(nextKey))
		return err
	}, func() error {
		_, err = s.Put(k, value)
		return err
	})
	return
}

func (s badgerStore[K, V]) Put(key K, value V) (success bool, err error) {
	err = s.withWrite(func(tx *badger.Txn) error {
		return s.PutWithTxn(tx, key, value)
	})
	return err == nil, err
}

func (s badgerStore[K, V]) PutWithTxn(tx *badger.Txn, key K, value V) (err error) {
	serializedKey, serializedValue, err := s.serializeKV(key, value)
	if err != nil {
		return
	}
	return tx.Set(serializedKey, serializedValue)
}

func (s badgerStore[K, V]) Update(key K, value V) (success bool, err error) {
	err = s.withWrite(func(tx *badger.Txn) error {
		return s.UpdateWithTxn(tx, key, value)
	})
	return err == nil, err
}

func (s badgerStore[K, V]) UpdateWithTxn(tx *badger.Txn, key K, value V) (err error) {
	serializedKey, serializedValue, err := s.serializeKV(key, value)
	if err != nil {
		return err
	}
	_, err = tx.Get(serializedKey)
	// update only when record exists
	if err == badger.ErrKeyNotFound {
		return errors.Error("record " + string(serializedKey) + " does not exist")
	}
	return tx.Set(serializedKey, serializedValue)
}

func (s badgerStore[K, V]) Delete(key K) (bool, error) {
	err := s.withWrite(func(tx *badger.Txn) error {
		return s.DeleteWithTx(tx, key)
	})
	return err != nil, err
}

func (s badgerStore[K, V]) DeleteWithTx(tx *badger.Txn, key K) error {
	serializedKey, err := s.KeySerializer(key)
	if err != nil {
		return err
	}
	return tx.Delete(serializedKey)

}

func (s badgerStore[K, V]) Query(filter func(key K, record V) bool) (res []V, err error) {
	err = s.iterate(func(k K, value V) error {
		if filter(k, value) {
			res = append(res, value)
		}
		return nil
	})
	return
}

func (s badgerStore[K, V]) Iterate(itr func(key K, record V)) error {
	return s.iterate(func(k K, value V) error {
		itr(k, value)
		return nil
	})
}

func (s badgerStore[K, V]) BulkGet(keys []K) (res []V, err error) {
	keySet := make(map[interface{}]bool)
	for _, key := range keys {
		keySet[key] = true
	}
	err = s.iterate(func(key K, value V) error {
		if keySet[key] {
			res = append(res, value)
		}
		return nil
	})
	return
}

func (s badgerStore[K, V]) BulkPut(bulk map[K]V) (success bool, err error) {
	err = s.withWrite(func(tx *badger.Txn) error {
		for key, value := range bulk {
			var serializedKey, serializedValue []byte
			utils.ProcessWithErrors(func() error {
				serializedKey, serializedValue, err = s.serializeKV(key, value)
				return err
			}, func() error {
				return tx.Set(serializedKey, serializedValue)
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	return
}

func (s badgerStore[K, V]) BulkAdd(entities []V) (entitiesWithIds map[K]V, err error) {
	var (
		nextKey uint64
		k       K
	)
	entitiesWithIds = make(map[K]V)
	s.withWrite(func(tx *badger.Txn) error {
		for _, entity := range entities {
			err = utils.ProcessWithErrors(func() error {
				if s.seq == nil {
					return errors.Error("BulkAdd can not be applied in non-auto-incr-id store")
				}
				return nil
			}, func() error {
				nextKey, err = s.seq.Next()
				if nextKey == 0 {
					nextKey, err = s.seq.Next()
				}
				return err
			}, func() error {
				k, err = s.KeyDeserializer(uint64ToBytes(nextKey))
				return err
			}, func() error {
				err = s.PutWithTxn(tx, k, entity)
				return err
			}, func() error {
				entitiesWithIds[k] = entity
				return nil
			})
			if err != nil {
				return err
			}
		}
		return err
	})
	return
}

func (s badgerStore[K, V]) WithTx(cb func(*badger.Txn) error) error {
	txn := s.db.NewTransaction(true)
	defer txn.Discard()
	return cb(txn)
}

func (s badgerStore[K, V]) iterate(cb func(k K, v V) error) error {
	return s.withRead(func(tx *badger.Txn) error {
		opt := badger.DefaultIteratorOptions
		itr := tx.NewIterator(opt)
		defer itr.Close()
		for itr.Rewind(); itr.Valid(); itr.Next() {
			item := itr.Item()
			rawKey := item.Key()
			var (
				key   K
				value V
				err   error
			)
			item.Value(func(val []byte) error {
				key, value, err = s.deserializeKV(rawKey, val)
				return err
			})
			err = cb(key, value)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (s badgerStore[K, V]) Close() error {
	return s.db.Close()
}

func (s badgerStore[K, V]) Drop() error {
	s.doGC()
	return s.db.DropAll()
}

func (s badgerStore[K, V]) serializeKV(key K, value V) (k, v []byte, e error) {
	var (
		zeroK K
		zeroV V
	)
	e = utils.ProcessWithErrors(func() error {
		if key != zeroK {
			k, e = s.KeySerializer(key)
		}
		return e
	}, func() error {
		if value != zeroV {
			v, e = s.ValueSerializer(value)
		}
		return e
	})
	return
}

func (s badgerStore[K, V]) deserializeKV(rawKey []byte, rawValue []byte) (key K, value V, e error) {
	e = utils.ProcessWithErrors(func() error {
		if rawKey != nil {
			key, e = s.KeyDeserializer(rawKey)
		}
		return e
	}, func() error {
		if rawValue != nil {
			value, e = s.ValueDeserializer(rawValue)
		}
		return e
	})
	return
}

type StringKVSerializationHandler struct{}

func NewStringKVSerializationHandler() BadgerStoreSerializeHandler[string, string] {
	return StringKVSerializationHandler{}
}

func (h StringKVSerializationHandler) KeySerializer(k string) ([]byte, error) {
	return []byte(k), nil
}

func (h StringKVSerializationHandler) KeyDeserializer(k []byte) (string, error) {
	return string(k), nil
}

func (h StringKVSerializationHandler) ValueSerializer(v string) ([]byte, error) {
	return []byte(v), nil
}

func (h StringKVSerializationHandler) ValueDeserializer(v []byte) (string, error) {
	return string(v), nil
}

func uint64ToBytes(i uint64) []byte {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], i)
	return buf[:]
}

func bytesToUint64(b []byte) uint64 {
	return binary.BigEndian.Uint64(b)
}
