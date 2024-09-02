package child

func (ch *BaseChild) InitializeTree(blockHeight uint64) bool {
	if ch.initializeTreeFn != nil {
		ok, err := ch.initializeTreeFn(blockHeight)
		if err != nil {
			panic("failed to initialize working tree: " + err.Error())
		}
		return ok
	}
	return false
}
