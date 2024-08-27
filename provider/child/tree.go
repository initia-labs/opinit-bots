package child

func (b *BaseChild) InitializeTree() (executed bool) {
	b.initializeTreeOnce.Do(func() {
		if b.initializeTreeFn != nil {
			executed = true
			err := b.initializeTreeFn()
			if err != nil {
				panic("failed to initialize working tree: " + err.Error())
			}
		}
	})
	return executed
}
