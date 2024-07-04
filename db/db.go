package db

import (
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

func (db *LevelDB) Set(key []byte, value []byte) error {
	return db.db.Put(append(db.prefix, key...), value, nil)
}

func (db *LevelDB) Get(key []byte) ([]byte, error) {
	return db.db.Get(append(db.prefix, key...), nil)
}

func (db *LevelDB) Delete(key []byte) error {
	return db.db.Delete(append(db.prefix, key...), nil)
}

func (db *LevelDB) Close() error {
	return db.db.Close()
}

func (db *LevelDB) Iterate(start, exclusiveEnd []byte, cb func(key, value []byte) (stop bool)) error {
	iter := db.db.NewIterator(&util.Range{append(db.prefix, start...), exclusiveEnd}, nil)
	for iter.Next() {
		if cb(iter.Key(), iter.Value()) {
			break
		}
	}
	iter.Release()
	return iter.Error()
}

func (db *LevelDB) WithPrefix(prefix []byte) types.DB {
	return &LevelDB{
		db:     db.db,
		prefix: append(db.prefix, prefix...),
	}
}
