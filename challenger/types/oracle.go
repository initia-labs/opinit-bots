package types

import "golang.org/x/crypto/sha3"

func OracleChecksum(data []byte) []byte {
	checksum := sha3.Sum256(data)
	return checksum[:]
}
