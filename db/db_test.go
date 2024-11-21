package db

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/storage"

	dbtypes "github.com/initia-labs/opinit-bots/db/types"
	"github.com/initia-labs/opinit-bots/types"
)

func makeTestSet() []types.KV {
	pairs := make([]types.KV, 0)

	pairs = append(pairs, types.KV{Key: []byte("key1"), Value: []byte("value1")})
	pairs = append(pairs, types.KV{Key: []byte("key2"), Value: []byte("value2")})
	pairs = append(pairs, types.KV{Key: []byte("key3"), Value: []byte("value3")})
	pairs = append(pairs, types.KV{Key: []byte("key4"), Value: []byte("value4")})
	pairs = append(pairs, types.KV{Key: []byte("key5"), Value: []byte("value5")})

	for i := range 1000 {
		pairs = append(pairs, types.KV{Key: append([]byte("key3"), dbtypes.FromUint64Key(uint64(i))...), Value: dbtypes.FromInt64(int64(i))})
	}

	for i := 5; i <= 5000; i += 5 {
		pairs = append(pairs, types.KV{Key: append([]byte("key4"), dbtypes.FromUint64Key(uint64(i))...), Value: dbtypes.FromInt64(int64(i))})
	}
	return pairs
}

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

func CreateTestDB(t *testing.T, pairs []types.KV) *LevelDB {
	db, err := NewMemDB()
	require.NoError(t, err)

	for _, pair := range pairs {
		err := db.Set(pair.Key, pair.Value)
		require.NoError(t, err)
	}
	return db
}

func TestNewDB(t *testing.T) {
	db, err := NewMemDB()
	require.NoError(t, err)
	require.NotNil(t, db)
}

func TestClose(t *testing.T) {
	db, err := NewMemDB()
	require.NoError(t, err)

	err = db.Close()
	require.NoError(t, err)

	err = db.Close()
	require.Error(t, err)
}

func TestPrefix(t *testing.T) {
	db, err := NewMemDB()
	require.NoError(t, err)

	require.Equal(t, db.prefix, []byte(nil))
	require.Equal(t, db.GetPrefix(), []byte(nil))

	db.prefix = []byte("abc")
	require.Equal(t, db.GetPrefix(), []byte("abc"))
}

func TestPrefixedKey(t *testing.T) {
	db, err := NewMemDB()
	require.NoError(t, err)

	prefixedKey := db.PrefixedKey([]byte("key1"))
	require.Equal(t, prefixedKey, []byte("/key1"))

	db.prefix = []byte("abc")
	prefixedKey = db.PrefixedKey([]byte("key1"))
	require.Equal(t, prefixedKey, []byte("abc/key1"))

	db.prefix = []byte("abc/def")
	prefixedKey = db.PrefixedKey([]byte("key2"))
	require.Equal(t, prefixedKey, []byte("abc/def/key2"))
}

func TestUnprefixedKey(t *testing.T) {
	db, err := NewMemDB()
	require.NoError(t, err)

	unprefixedKey := db.UnprefixedKey([]byte("key1"))
	require.Equal(t, unprefixedKey, []byte("key1"))

	unprefixedKey = db.UnprefixedKey([]byte("/key1"))
	require.Equal(t, unprefixedKey, []byte("key1"))

	db.prefix = []byte("abc")
	unprefixedKey = db.UnprefixedKey([]byte("abc/key1"))
	require.Equal(t, unprefixedKey, []byte("key1"))

	db.prefix = []byte("abc/def")
	unprefixedKey = db.UnprefixedKey([]byte("abc/def/key2"))
	require.Equal(t, unprefixedKey, []byte("key2"))
}

func TestWithPrefix(t *testing.T) {
	db, err := NewMemDB()
	require.NoError(t, err)

	newDB := db.WithPrefix([]byte("abc"))
	require.Equal(t, newDB.GetPrefix(), []byte("/abc"))

	newDB2 := newDB.WithPrefix([]byte("abc"))
	require.Equal(t, newDB2.GetPrefix(), []byte("/abc/abc"))

	newDB3 := newDB2.WithPrefix([]byte("abc/def"))
	require.Equal(t, newDB3.GetPrefix(), []byte("/abc/abc/abc/def"))
}

func TestSet(t *testing.T) {
	db, err := NewMemDB()
	require.NoError(t, err)

	cases := []struct {
		title    string
		key      []byte
		value    []byte
		expected bool
	}{
		{"simple set", []byte("key1"), []byte("value1"), true},
		{"duplicated key", []byte("key1"), []byte("value2"), true},
		{"empty key", []byte(""), []byte("value1"), true},
		{"empty value", []byte("key2"), []byte(""), true},
	}

	for _, tc := range cases {
		t.Run(tc.title, func(t *testing.T) {
			err := db.Set(tc.key, tc.value)
			if tc.expected {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestGet(t *testing.T) {
	db, err := NewMemDB()
	require.NoError(t, err)

	cases := []struct {
		title    string
		key      []byte
		value    []byte
		expected []bool
	}{
		{"simple get", []byte("key1"), []byte("value1"), []bool{false, true, true}},
		{"empty key", []byte(""), []byte("value1"), []bool{false, true, true}},
		{"empty value", []byte("key2"), nil, []bool{false, true, true}},
	}

	for _, tc := range cases {
		t.Run(tc.title, func(t *testing.T) {
			_, err := db.Get(tc.key)
			if tc.expected[0] {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}

			err = db.Set(tc.key, tc.value)
			if tc.expected[1] {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}

			value, err := db.Get(tc.key)
			if tc.expected[2] {
				require.NoError(t, err)
				require.Equal(t, tc.value, value)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	db, err := NewMemDB()
	require.NoError(t, err)

	cases := []struct {
		title    string
		key      []byte
		value    []byte
		expected bool
	}{
		{"simple delete", []byte("key1"), []byte("value1"), true},
		{"not existing key", []byte("key1"), nil, true},
	}

	for _, tc := range cases {
		t.Run(tc.title, func(t *testing.T) {
			if tc.value != nil {
				err = db.Set(tc.key, tc.value)
				require.NoError(t, err)
			}

			err := db.Delete(tc.key)
			if tc.expected {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestBatchSet(t *testing.T) {
	db, err := NewMemDB()
	require.NoError(t, err)

	pairs := makeTestSet()

	err = db.BatchSet(pairs...)
	require.NoError(t, err)

	for _, pair := range pairs {
		value, err := db.Get(pair.Key)
		require.NoError(t, err)
		require.Equal(t, pair.Value, value)
	}

	err = db.BatchSet([]types.KV{}...)
	require.NoError(t, err)
}

func TestIterate(t *testing.T) {
	pairs := makeTestSet()
	db := CreateTestDB(t, pairs)

	cases := []struct {
		title    string
		prefix   []byte
		start    []byte
		expected int
	}{
		{"empty prefix", nil, nil, 2005},
		{"prefix key3", []byte("key3"), nil, 1001},
		{"prefix key3 start 1", []byte("key3"), append([]byte("key3"), dbtypes.FromUint64Key(1)...), 999},
		{"prefix key3 start 500", []byte("key3"), append([]byte("key3"), dbtypes.FromUint64Key(500)...), 500},
		{"prefix key3 start 999", []byte("key3"), append([]byte("key3"), dbtypes.FromUint64Key(999)...), 1},
		{"prefix key3 start 1000", []byte("key3"), append([]byte("key3"), dbtypes.FromUint64Key(1000)...), 0},
		{"prefix key4 start 1001", []byte("key4"), append([]byte("key4"), dbtypes.FromUint64Key(1001)...), 800},
	}

	for _, tc := range cases {
		t.Run(tc.title, func(t *testing.T) {
			i := 0
			err := db.Iterate(tc.prefix, tc.start, func(key, value []byte) (stop bool, err error) {
				i++
				return false, nil
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, i)
		})
	}
}

func TestIterateError(t *testing.T) {
	pairs := makeTestSet()
	db := CreateTestDB(t, pairs)

	err := db.Iterate(nil, nil, func(key, value []byte) (stop bool, err error) {
		return false, nil
	})
	require.NoError(t, err)

	err = db.Iterate(nil, nil, func(key, value []byte) (stop bool, err error) {
		return true, nil
	})
	require.NoError(t, err)

	err = db.Iterate(nil, nil, func(key, value []byte) (stop bool, err error) {
		return false, errors.New("simple error")
	})
	require.Error(t, err)
}

func TestReverseIterate(t *testing.T) {
	pairs := makeTestSet()
	db := CreateTestDB(t, pairs)

	cases := []struct {
		title    string
		prefix   []byte
		start    []byte
		expected int
	}{
		{"empty prefix", nil, nil, 2005},
		{"prefix key3", []byte("key3"), nil, 1001},
		{"prefix key3 start 1", []byte("key3"), append([]byte("key3"), dbtypes.FromUint64Key(1)...), 3},
		{"prefix key3 start 500", []byte("key3"), append([]byte("key3"), dbtypes.FromUint64Key(500)...), 502},
		{"prefix key3 start 999", []byte("key3"), append([]byte("key3"), dbtypes.FromUint64Key(999)...), 1001},
		{"prefix key3 start 1000", []byte("key3"), append([]byte("key3"), dbtypes.FromUint64Key(1000)...), 1001},
		{"prefix key4 start 3", []byte("key4"), append([]byte("key4"), dbtypes.FromUint64Key(3)...), 1},
		{"prefix key4 start 4997", []byte("key4"), append([]byte("key4"), dbtypes.FromUint64Key(4997)...), 1000},
		{"prefix key4 start 1000", []byte("key4"), append([]byte("key4"), dbtypes.FromUint64Key(1000)...), 201},
	}

	for _, tc := range cases {
		t.Run(tc.title, func(t *testing.T) {
			i := 0
			err := db.ReverseIterate(tc.prefix, tc.start, func(key, value []byte) (stop bool, err error) {
				i++
				return false, nil
			})
			require.NoError(t, err)
			require.Equal(t, tc.expected, i)
		})
	}
}

func TestSeekPrevInclusiveKey(t *testing.T) {
	pairs := makeTestSet()
	db := CreateTestDB(t, pairs)

	cases := []struct {
		title    string
		prefix   []byte
		start    []byte
		expected []byte
		err      bool
	}{
		{"prefix key0", []byte("key0"), nil, []byte("key0"), true},
		{"prefix key3 start 1", []byte("key3"), append([]byte("key3"), dbtypes.FromUint64Key(1)...), append([]byte("key3"), dbtypes.FromUint64Key(1)...), false},
		{"prefix key3 start 500", []byte("key3"), append([]byte("key3"), dbtypes.FromUint64Key(500)...), append([]byte("key3"), dbtypes.FromUint64Key(500)...), false},
		{"prefix key3 start 1000", []byte("key3"), append([]byte("key3"), dbtypes.FromUint64Key(1000)...), append([]byte("key3"), dbtypes.FromUint64Key(999)...), false},
		{"prefix key4 start 0", []byte("key4"), append([]byte("key4"), dbtypes.FromUint64Key(0)...), []byte("key4"), false},
		{"prefix key4 start 4997", []byte("key4"), append([]byte("key4"), dbtypes.FromUint64Key(4997)...), append([]byte("key4"), dbtypes.FromUint64Key(4995)...), false},
		{"prefix key4 start 1000", []byte("key4"), append([]byte("key4"), dbtypes.FromUint64Key(1000)...), append([]byte("key4"), dbtypes.FromUint64Key(1000)...), false},
		{"prefix key4 start 6000", []byte("key4"), append([]byte("key4"), dbtypes.FromUint64Key(6000)...), append([]byte("key4"), dbtypes.FromUint64Key(5000)...), false},
	}

	for _, tc := range cases {
		t.Run(tc.title, func(t *testing.T) {
			key, _, err := db.SeekPrevInclusiveKey(tc.prefix, tc.start)
			if tc.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, key)
			}
		})
	}
}

func TestNewStage(t *testing.T) {
	db, err := NewMemDB()
	require.NoError(t, err)

	tstage := db.NewStage()
	require.NotNil(t, tstage)

	stage, ok := tstage.(*Stage)
	require.True(t, ok)
	require.Equal(t, stage.batch.Len(), 0)
	require.Equal(t, len(stage.kvmap), 0)
	require.Equal(t, stage.parent, db)
}
