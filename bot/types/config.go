package types

type Config interface {
	Validate() error
}
