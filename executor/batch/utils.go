package batch

import "encoding/binary"

func WriteBatch(data []byte) []byte {
	lengthBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(lengthBytes, uint64(len(lengthBytes)))
	return append(lengthBytes, data...)
}
