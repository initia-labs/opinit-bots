package child

func (b *BaseChild) InitializeTree(blockHeight uint64) bool {
	if b.initializeTreeFn != nil {
		ok, err := b.initializeTreeFn(blockHeight)
		if err != nil {
			panic("failed to initialize working tree: " + err.Error())
		}
		return ok
	}
	return false
}
