package types

// KV is a key-value pair with prefixing the key.
type KV struct {
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

	BatchSet(...KV) error
	Iterate([]byte, []byte, func([]byte, []byte) (bool, error)) error
	ReverseIterate([]byte, []byte, func([]byte, []byte) (bool, error)) error
	SeekPrevInclusiveKey([]byte, []byte) ([]byte, []byte, error)
	WithPrefix([]byte) DB
	PrefixedKey([]byte) []byte
	UnprefixedKey([]byte) []byte
	GetPrefix() []byte
}

type CommitDB interface {
	BasicDB

	WithPrefixedKey(prefixedKey func(key []byte) []byte) CommitDB
	Commit() error
	Reset()
	Len() int
}
