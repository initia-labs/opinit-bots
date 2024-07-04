package types

type DB interface {
	Get([]byte) ([]byte, error)
	Set([]byte, []byte) error
	Delete([]byte) error
	Close() error
	Iterate([]byte, []byte, func([]byte, []byte) bool) error
	WithPrefix([]byte) DB
}
