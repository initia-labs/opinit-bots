package db

import (
	dbtypes "github.com/initia-labs/opinit-bots-go/db/types"
	"github.com/initia-labs/opinit-bots-go/types"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

var _ types.DB = (*LevelDB)(nil)

type LevelDB struct {
	db     *leveldb.DB
	prefix []byte
}

func NewDB(path string) (types.DB, error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}

	return &LevelDB{
		db: db,
	}, nil
}

func (db *LevelDB) RawBatchSet(kvs ...types.KV) error {
	if len(kvs) == 0 {
		return nil
	}
	batch := new(leveldb.Batch)
	for _, kv := range kvs {
		if kv.Value == nil {
			batch.Delete(kv.Key)
		} else {
			batch.Put(kv.Key, kv.Value)
		}
	}
	return db.db.Write(batch, nil)
}

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

func (db *LevelDB) Set(key []byte, value []byte) error {
	return db.db.Put(db.PrefixedKey(key), value, nil)
}

func (db *LevelDB) Get(key []byte) ([]byte, error) {
	return db.db.Get(db.PrefixedKey(key), nil)
}

func (db *LevelDB) Delete(key []byte) error {
	return db.db.Delete(db.PrefixedKey(key), nil)
}

func (db *LevelDB) Close() error {
	return db.db.Close()
}

func (db *LevelDB) PrefixedIterate(prefix []byte, cb func(key, value []byte) (stop bool)) error {
	iter := db.db.NewIterator(util.BytesPrefix(db.PrefixedKey(prefix)), nil)
	for iter.Next() {
		key := db.UnprefixedKey(iter.Key())
		if cb(key, iter.Value()) {
			break
		}
	}
	iter.Release()
	return iter.Error()
}

func (db *LevelDB) SeekPrevInclusiveKey(prefix []byte, key []byte) (k []byte, v []byte, err error) {
	iter := db.db.NewIterator(util.BytesPrefix(db.PrefixedKey(prefix)), nil)
	if iter.Seek(db.PrefixedKey(key)) || iter.Valid() && iter.Prev() {
		k = db.UnprefixedKey(iter.Key())
		v = iter.Value()
	} else {
		err = dbtypes.ErrNotFound
	}
	iter.Release()
	if iter.Error() != nil {
		err = iter.Error()
	}
	return k, v, err
}

func (db *LevelDB) WithPrefix(prefix []byte) types.DB {
	return &LevelDB{
		db:     db.db,
		prefix: db.PrefixedKey(prefix),
	}
}

func (db LevelDB) PrefixedKey(key []byte) []byte {
	return append(append(db.prefix, []byte("/")...), key...)
}

func (db LevelDB) UnprefixedKey(key []byte) []byte {
	return key[len(db.prefix)+1:]
}
