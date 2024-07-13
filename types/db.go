package types

type KV struct {
	Key   []byte
	Value []byte
}

type DB interface {
	Get([]byte) ([]byte, error)
	Set([]byte, []byte) error
	RawBatchSet(...KV) error
	BatchSet(...KV) error
	Delete([]byte) error
	Close() error
	Iterate([]byte, []byte, func([]byte, []byte) bool) error
	SeekPrevInclusiveKey([]byte) ([]byte, []byte, error)
	WithPrefix([]byte) DB
	PrefixedKey([]byte) []byte
	UnprefixedKey([]byte) []byte
}
