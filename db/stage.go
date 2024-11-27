package db

import (
	dbtypes "github.com/initia-labs/opinit-bots/db/types"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/syndtr/goleveldb/leveldb"
	"golang.org/x/exp/maps"
)

type Stage struct {
	batch  leveldb.Batch
	kvmap  map[string][]byte
	parent *LevelDB
}

var _ types.CommitDB = (*Stage)(nil)

func (s *Stage) Set(key []byte, value []byte) error {
	prefixedKey := s.parent.PrefixedKey(key)
	s.batch.Put(prefixedKey, value)
	s.kvmap[string(prefixedKey)] = value
	return nil
}

func (s Stage) Get(key []byte) ([]byte, error) {
	prefixedKey := s.parent.PrefixedKey(key)
	value, ok := s.kvmap[string(prefixedKey)]
	if ok {
		return value, nil
	}
	return s.parent.Get(key)
}

func (s *Stage) Delete(key []byte) error {
	prefixedKey := s.parent.PrefixedKey(key)
	s.batch.Delete(prefixedKey)
	s.kvmap[string(prefixedKey)] = nil
	return nil
}

func (s *Stage) Commit() error {
	err := s.parent.db.Write(&s.batch, nil)
	if err != nil {
		return err
	}
	return nil
}

func (s *Stage) ExecuteFnWithDB(db types.DB, fn func() error) error {
	existing := s.parent
	defer func() {
		s.parent = existing
	}()

	leveldb, ok := db.(*LevelDB)
	if !ok {
		return dbtypes.ErrInvalidParentDBType
	}
	s.parent = leveldb

	return fn()
}

func (s Stage) Len() int {
	return s.batch.Len()
}

func (s *Stage) Reset() {
	s.batch.Reset()
	maps.Clear(s.kvmap)
}
