package db

import "github.com/initia-labs/opinit-bots/types"

type Stage struct {
	batch  []types.RawKV
	kvmap  map[string][]byte
	parent types.DB
}

var _ types.CommitDB = (*Stage)(nil)

func (s *Stage) Set(key []byte, value []byte) error {
	s.batch = append(s.batch, types.RawKV{
		Key:   key,
		Value: value,
	})
	s.kvmap[string(key)] = value
	return nil
}

func (s Stage) Get(key []byte) ([]byte, error) {
	value := s.kvmap[string(key)]
	if value != nil {
		return value, nil
	}
	return s.parent.Get(key)
}

func (s *Stage) Delete(key []byte) error {
	s.batch = append(s.batch, types.RawKV{
		Key:   key,
		Value: nil,
	})
	s.kvmap[string(key)] = nil
	return nil
}

func (s *Stage) Commit() error {
	return s.parent.RawBatchSet(s.batch...)
}
