package db

import (
	"bytes"

	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"

	dbtypes "github.com/initia-labs/opinit-bots/db/types"
	"github.com/initia-labs/opinit-bots/types"
)

var _ types.DB = (*LevelDB)(nil)

type LevelDB struct {
	db     *leveldb.DB
	prefix []byte
}

func NewDB(path string) (*LevelDB, error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}

	return &LevelDB{
		db: db,
	}, nil
}

// BatchSet sets the key-value pairs in the database with prefixing the keys.
func (db *LevelDB) BatchSet(kvs ...types.KV) error {
	if len(kvs) == 0 {
		return nil
	}
	batch := new(leveldb.Batch)
	for _, kv := range kvs {
		if kv.Value == nil {
			batch.Delete(db.PrefixedKey(kv.Key))
		} else {
			batch.Put(db.PrefixedKey(kv.Key), kv.Value)
		}
	}
	return db.db.Write(batch, nil)
}

// Set sets the key-value pair in the database with prefixing the key.
func (db *LevelDB) Set(key []byte, value []byte) error {
	return db.db.Put(db.PrefixedKey(key), value, nil)
}

// Get gets the value of the key in the database with prefixing the key.
func (db *LevelDB) Get(key []byte) ([]byte, error) {
	v, err := db.db.Get(db.PrefixedKey(key), nil)
	if errors.Is(err, leveldb.ErrNotFound) {
		return nil, errors.Wrapf(err, "key: %v", key)
	}
	return v, err
}

// Delete deletes the key in the database with prefixing the key.
func (db *LevelDB) Delete(key []byte) error {
	return db.db.Delete(db.PrefixedKey(key), nil)
}

// Close closes the database.
func (db *LevelDB) Close() error {
	return db.db.Close()
}

// Iterate iterates over the key-value pairs in the database with prefixing the keys.
//
// @dev: `LevelDB.prefix + prefix` is used as the prefix for the iteration.
func (db *LevelDB) Iterate(prefix []byte, start []byte, cb func(key, value []byte) (stop bool, err error)) (iterErr error) {
	iter := db.db.NewIterator(util.BytesPrefix(db.PrefixedKey(prefix)), nil)
	defer func() {
		iter.Release()
		if iterErr == nil {
			iterErr = iter.Error()
		}
	}()

	if start != nil {
		iter.Seek(db.PrefixedKey(start))
	} else {
		iter.First()
	}

	for iter.Valid() {
		key := db.UnprefixedKey(bytes.Clone(iter.Key()))
		if stop, err := cb(key, bytes.Clone(iter.Value())); err != nil {
			return err
		} else if stop {
			break
		}
		iter.Next()
	}
	return
}

// ReverseIterate iterates over the key-value pairs in the database with prefixing the keys in reverse order.
func (db *LevelDB) ReverseIterate(prefix []byte, start []byte, cb func(key, value []byte) (stop bool, err error)) (iterErr error) {
	iter := db.db.NewIterator(util.BytesPrefix(db.PrefixedKey(prefix)), nil)
	defer func() {
		iter.Release()
		if iterErr == nil {
			iterErr = iter.Error()
		}
	}()

	if start != nil {
		if ok := iter.Seek(db.PrefixedKey(start)); ok || iter.Last() {
			// if the valid key is not found, the iterator will be at the last key
			// if the key is found, the iterator will be at the key
			// or the previous key if the key is not found
			if ok && !bytes.Equal(db.PrefixedKey(start), iter.Key()) {
				iter.Prev()
			}
		}
	} else {
		iter.Last()
	}

	for iter.Valid() {
		key := db.UnprefixedKey(bytes.Clone(iter.Key()))
		if stop, err := cb(key, bytes.Clone(iter.Value())); err != nil {
			return err
		} else if stop {
			break
		}

		iter.Prev()
	}
	return
}

// SeekPrevInclusiveKey seeks the previous key-value pair in the database with prefixing the keys.
// If the key is found, it returns the key-value pair.
//
// @dev: `LevelDB.prefix + prefix` is used as the prefix for the iteration.
func (db *LevelDB) SeekPrevInclusiveKey(prefix []byte, key []byte) (k []byte, v []byte, err error) {
	iter := db.db.NewIterator(util.BytesPrefix(db.PrefixedKey(prefix)), nil)
	defer func() {
		iter.Release()
		if err == nil {
			err = iter.Error()
		}
	}()
	if ok := iter.Seek(db.PrefixedKey(key)); ok || iter.Last() {
		// if the valid key is not found, the iterator will be at the last key
		// if the key is found, the iterator will be at the key
		// or the previous key if the key is not found
		if ok && !bytes.Equal(db.PrefixedKey(key), iter.Key()) {
			iter.Prev()
		}
		k = db.UnprefixedKey(bytes.Clone(iter.Key()))
		v = bytes.Clone(iter.Value())
	} else {
		err = dbtypes.ErrNotFound
	}
	return k, v, err
}

// WithPrefix returns a new LevelDB with the given prefix.
func (db *LevelDB) WithPrefix(prefix []byte) types.DB {
	return &LevelDB{
		db:     db.db,
		prefix: db.PrefixedKey(prefix),
	}
}

// PrefixedKey prefixes the key with the LevelDB.prefix.
func (db LevelDB) PrefixedKey(key []byte) []byte {
	return append(append(db.prefix, dbtypes.Splitter), key...)
}

// UnprefixedKey remove the prefix from the key, only
// if the key has the prefix.
func (db LevelDB) UnprefixedKey(key []byte) []byte {
	return bytes.TrimPrefix(key, append(db.prefix, dbtypes.Splitter))
}

func (db LevelDB) GetPrefix() []byte {
	return db.prefix
}

func (db *LevelDB) NewStage() types.CommitDB {
	return newStage(db)
}
