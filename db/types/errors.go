package types

import (
	"errors"

	"github.com/syndtr/goleveldb/leveldb"
)

var ErrNotFound = leveldb.ErrNotFound
var ErrInvalidParentDBType = errors.New("invalid parent DB type")
