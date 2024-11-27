package db

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
)

func CreateTestStage(t *testing.T, db *LevelDB) (*LevelDB, *Stage, error) {
	var err error
	if db == nil {
		db, err = NewMemDB()
		require.NoError(t, err)
	}
	tstage := db.NewStage()
	stage, ok := tstage.(*Stage)
	require.True(t, ok)
	return db, stage, nil
}

func TestStageReset(t *testing.T) {
	_, stage, err := CreateTestStage(t, nil)
	require.NoError(t, err)

	stage.Reset()
	require.Equal(t, stage.batch.Len(), 0)
	require.Equal(t, len(stage.kvmap), 0)

	err = stage.Set([]byte("key"), []byte("value"))
	require.NoError(t, err)
	require.Equal(t, stage.batch.Len(), 1)
	require.Equal(t, len(stage.kvmap), 1)

	stage.Reset()
	require.Equal(t, stage.batch.Len(), 0)
	require.Equal(t, len(stage.kvmap), 0)
}

func TestStageLen(t *testing.T) {
	_, stage, err := CreateTestStage(t, nil)
	require.NoError(t, err)

	stage.Reset()
	require.Equal(t, stage.Len(), 0)

	err = stage.Set([]byte("key"), []byte("value"))
	require.NoError(t, err)
	require.Equal(t, stage.Len(), 1)

	stage.Reset()
	require.Equal(t, stage.Len(), 0)
}

func TestStageSet(t *testing.T) {
	_, stage, err := CreateTestStage(t, nil)
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
			err := stage.Set(tc.key, tc.value)
			if tc.expected {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestStageGet(t *testing.T) {
	_, stage, err := CreateTestStage(t, nil)
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
			_, err := stage.Get(tc.key)
			if tc.expected[0] {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}

			err = stage.Set(tc.key, tc.value)
			if tc.expected[1] {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}

			value, err := stage.Get(tc.key)
			if tc.expected[2] {
				require.NoError(t, err)
				require.Equal(t, tc.value, value)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestStageDelete(t *testing.T) {
	_, stage, err := CreateTestStage(t, nil)
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
				err = stage.Set(tc.key, tc.value)
				require.NoError(t, err)
			}

			err := stage.Delete(tc.key)
			if tc.expected {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestStageCommit(t *testing.T) {
	db := CreateTestDB(t, makeTestSet())
	_, stage, err := CreateTestStage(t, db)
	require.NoError(t, err)

	err = stage.Delete([]byte("key1"))
	require.NoError(t, err)

	err = stage.Delete([]byte("key1"))
	require.NoError(t, err)

	err = stage.Set([]byte("key2"), []byte("new_value2"))
	require.NoError(t, err)

	err = stage.Set([]byte("key10"), []byte("value10"))
	require.NoError(t, err)

	err = stage.Commit()
	require.NoError(t, err)
	stage.Reset()

	_, err = db.Get([]byte("key1"))
	require.Error(t, err)
	value, err := db.Get([]byte("key2"))
	require.NoError(t, err)
	require.Equal(t, []byte("new_value2"), value)

	_, err = stage.Get([]byte("key1"))
	require.Error(t, err)
	value, err = stage.Get([]byte("key2"))
	require.NoError(t, err)
	require.Equal(t, []byte("new_value2"), value)

	value, err = stage.Get([]byte("key10"))
	require.NoError(t, err)
	require.Equal(t, []byte("value10"), value)

	err = stage.Delete([]byte("key11"))
	require.NoError(t, err)

	err = stage.Commit()
	require.NoError(t, err)

	_, err = db.Get([]byte("key11"))
	require.Error(t, err)
}

func TestExecuteFnWithDB(t *testing.T) {
	db, err := NewMemDB()
	require.NoError(t, err)

	tdb1 := db.WithPrefix([]byte("123"))
	db1 := tdb1.(*LevelDB)
	db2 := db.WithPrefix([]byte("456"))

	_, stage1, err := CreateTestStage(t, db1)
	require.NoError(t, err)

	err = stage1.Set([]byte("key1"), []byte("value1"))
	require.NoError(t, err)

	err = stage1.ExecuteFnWithDB(db2, func() error {
		err := stage1.Set([]byte("key1"), []byte("value2"))
		require.NoError(t, err)
		return err
	})
	require.NoError(t, err)

	require.Equal(t, stage1.kvmap[string(db1.PrefixedKey([]byte("key1")))], []byte("value1"))
	require.Equal(t, stage1.kvmap[string(db2.PrefixedKey([]byte("key1")))], []byte("value2"))
}

func TestStageAll(t *testing.T) {
	db, err := NewMemDB()
	require.NoError(t, err)

	tdb1 := db.WithPrefix([]byte("123"))
	db1 := tdb1.(*LevelDB)
	db2 := db.WithPrefix([]byte("456"))

	_, stage1, err := CreateTestStage(t, db1)
	require.NoError(t, err)

	err = stage1.Set([]byte("key1"), []byte("value1"))
	require.NoError(t, err)

	err = stage1.ExecuteFnWithDB(db2, func() error {
		err := stage1.Set([]byte("key1"), []byte("value2"))
		require.NoError(t, err)
		return err
	})
	require.NoError(t, err)

	allKVs := stage1.All()
	require.Equal(t, len(allKVs), 2)
	require.Equal(t, allKVs["/123/key1"], []byte("value1"))
	require.Equal(t, allKVs["/456/key1"], []byte("value2"))

	maps.Clear(allKVs)
	require.Empty(t, allKVs)

	allKVs = stage1.All()
	require.Equal(t, len(allKVs), 2)
}
