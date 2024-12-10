package db

import (
	"github.com/initia-labs/opinit-bots/types"
	"github.com/syndtr/goleveldb/leveldb"
	"golang.org/x/exp/maps"
)

type Stage struct {
	batch  *leveldb.Batch
	kvmap  map[string][]byte
	parent *LevelDB

	prefixedKey func(key []byte) []byte
}

func newStage(parent *LevelDB) Stage {
	return Stage{
		batch:  new(leveldb.Batch),
		kvmap:  make(map[string][]byte),
		parent: parent,

		prefixedKey: parent.PrefixedKey,
	}
}

func (s Stage) WithPrefixedKey(prefixedKey func(key []byte) []byte) types.CommitDB {
	s.prefixedKey = prefixedKey
	return s
}

var _ types.CommitDB = Stage{}

func (s Stage) Set(key []byte, value []byte) error {
	prefixedKey := s.prefixedKey(key)
	s.batch.Put(prefixedKey, value)
	s.kvmap[string(prefixedKey)] = value
	return nil
}

func (s Stage) Get(key []byte) ([]byte, error) {
	prefixedKey := s.prefixedKey(key)
	value, ok := s.kvmap[string(prefixedKey)]
	if ok {
		return value, nil
	}
	return s.parent.Get(key)
}

func (s Stage) Delete(key []byte) error {
	prefixedKey := s.prefixedKey(key)
	s.batch.Delete(prefixedKey)
	s.kvmap[string(prefixedKey)] = nil
	return nil
}

func (s Stage) Commit() error {
	err := s.parent.db.Write(s.batch, nil)
	if err != nil {
		return err
	}
	return nil
}

func (s Stage) Len() int {
	return s.batch.Len()
}

func (s Stage) Reset() {
	s.batch.Reset()
	maps.Clear(s.kvmap)
}

func (s Stage) All() map[string][]byte {
	return maps.Clone(s.kvmap)
}
