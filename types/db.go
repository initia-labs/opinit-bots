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

type DB interface {
	Get([]byte) ([]byte, error)
	Set([]byte, []byte) error
	RawBatchSet(...RawKV) error
	BatchSet(...KV) error
	Delete([]byte) error
	Close() error
	PrefixedIterate([]byte, func([]byte, []byte) (bool, error)) error
	SeekPrevInclusiveKey([]byte, []byte) ([]byte, []byte, error)
	WithPrefix([]byte) DB
	PrefixedKey([]byte) []byte
	UnprefixedKey([]byte) []byte
}
