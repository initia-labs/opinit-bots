package types

import "errors"

var ErrKeyNotSet = errors.New("key not set")
var ErrAccountSequenceMismatch = errors.New("account sequence mismatch")
var ErrTxNotFound = errors.New("tx not found")
