package db

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/storage"
)

func NewMemDB() (*LevelDB, error) {
	s := storage.NewMemStorage()

	db, err := leveldb.Open(s, nil)
	if err != nil {
		return nil, err
	}

	return &LevelDB{
		db: db,
	}, nil
}
