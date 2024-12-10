package child

import "github.com/pkg/errors"

func (b *BaseChild) InitializeTree(blockHeight int64) bool {
	if b.initializeTreeFn != nil {
		ok, err := b.initializeTreeFn(blockHeight)
		if err != nil {
			panic(errors.Wrap(err, "failed to initialize working tree"))
		}
		return ok
	}
	return false
}
