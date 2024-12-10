package types

// Result of block bulk
type ResultBlockBulk struct {
	Blocks [][]byte `json:"blocks"`
}

// Result of raw commit bytes
type ResultRawCommit struct {
	Commit []byte `json:"commit"`
}
