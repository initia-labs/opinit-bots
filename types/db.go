package types

// KV is a key-value pair with prefixing the key.
type KV struct {
	Key   []byte
	Value []byte
}

// RawKV is a key-value pair without prefixing the key.
type RawKV struct {
	Key   []byte
	Value []byte
}

type BasicDB interface {
	Get([]byte) ([]byte, error)
	Set([]byte, []byte) error
	Delete([]byte) error
}

type DB interface {
	BasicDB

	NewStage() CommitDB

	RawBatchSet(...RawKV) error
	BatchSet(...KV) error
	PrefixedIterate([]byte, []byte, func([]byte, []byte) (bool, error)) error
	PrefixedReverseIterate([]byte, []byte, func([]byte, []byte) (bool, error)) error
	SeekPrevInclusiveKey([]byte, []byte) ([]byte, []byte, error)
	WithPrefix([]byte) DB
	PrefixedKey([]byte) []byte
	UnprefixedKey([]byte) []byte
	GetPrefix() []byte
}

type CommitDB interface {
	BasicDB

	ExecuteFnWithDB(DB, func() error) error
	Commit() error
	Reset()
}
