package types

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"time"

	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type DANode interface {
	Start(context.Context)
	HasKey() bool
	CreateBatchMsg([]byte) (sdk.Msg, error)
	BroadcastMsgs(btypes.ProcessedMsgs)
	ProcessedMsgsToRawKV(processedMsgs []btypes.ProcessedMsgs, delete bool) ([]types.RawKV, error)
	GetNodeStatus() nodetypes.Status
}

type LocalBatchInfo struct {
	// start l2 block height which is included in the batch
	Start int64 `json:"start"`
	// last l2 block height which is included in the batch
	End int64 `json:"end"`

	LastSubmissionTime time.Time `json:"last_submission_time"`
	BatchFileSize      int64     `json:"batch_size"`
}

type BatchDataType uint8

const (
	BatchDataTypeHeader BatchDataType = iota
	BatchDataTypeChunk
)

type BatchDataHeader struct {
	Start     uint64
	End       uint64
	Checksums [][]byte
}

type BatchDataChunk struct {
	Start     uint64
	End       uint64
	Index     uint64
	Length    uint64
	ChunkData []byte
}

func GetChecksumFromChunk(chunk []byte) [32]byte {
	return sha256.Sum256(chunk)
}

func MarshalBatchDataHeader(
	start uint64,
	end uint64,
	checksums [][]byte,
) []byte {
	data := make([]byte, 1)
	data[0] = byte(BatchDataTypeHeader)
	data = binary.BigEndian.AppendUint64(data, start)
	data = binary.BigEndian.AppendUint64(data, end)
	data = binary.BigEndian.AppendUint64(data, uint64(len(checksums)))
	for _, checksum := range checksums {
		data = append(data, checksum...)
	}
	return data
}

func UnmarshalBatchDataHeader(data []byte) (BatchDataHeader, error) {
	if len(data) < 25 {
		err := fmt.Errorf("invalid data length: %d, expected > 25", len(data))
		return BatchDataHeader{}, err
	}
	start := binary.BigEndian.Uint64(data[1:9])
	end := binary.BigEndian.Uint64(data[9:17])
	if start > end {
		return BatchDataHeader{}, fmt.Errorf("invalid start: %d, end: %d", start, end)
	}

	length := binary.BigEndian.Uint64(data[17:25])
	expectedLength, err := types.SafeInt64ToUint64(int64(len(data)-25) / 32)
	if err != nil {
		return BatchDataHeader{}, err
	}

	if int64(len(data)-25)%32 != 0 || expectedLength != length {
		err := fmt.Errorf("invalid checksum length: %d, data length: %d", length, len(data)-25)
		return BatchDataHeader{}, err
	}

	checksums := make([][]byte, 0, length)
	for i := 25; i < len(data); i += 32 {
		checksums = append(checksums, data[i:i+32])
	}

	return BatchDataHeader{
		Start:     start,
		End:       end,
		Checksums: checksums,
	}, nil
}

func MarshalBatchDataChunk(
	start uint64,
	end uint64,
	index uint64,
	length uint64,
	chunkData []byte,
) []byte {
	data := make([]byte, 1)
	data[0] = byte(BatchDataTypeChunk)
	data = binary.BigEndian.AppendUint64(data, start)
	data = binary.BigEndian.AppendUint64(data, end)
	data = binary.BigEndian.AppendUint64(data, index)
	data = binary.BigEndian.AppendUint64(data, length)
	data = append(data, chunkData...)
	return data
}

func UnmarshalBatchDataChunk(data []byte) (BatchDataChunk, error) {
	if len(data) < 33 {
		err := fmt.Errorf("invalid data length: %d, expected > 33", len(data))
		return BatchDataChunk{}, err
	}
	start := binary.BigEndian.Uint64(data[1:9])
	end := binary.BigEndian.Uint64(data[9:17])
	if start > end {
		return BatchDataChunk{}, fmt.Errorf("invalid start: %d, end: %d", start, end)
	}
	index := binary.BigEndian.Uint64(data[17:25])
	length := binary.BigEndian.Uint64(data[25:33])
	chunkData := data[33:]

	return BatchDataChunk{
		Start:     start,
		End:       end,
		Index:     index,
		Length:    length,
		ChunkData: chunkData,
	}, nil
}
