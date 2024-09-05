package types

import (
	"context"
	"encoding/binary"
	"errors"
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
	Start uint64 `json:"start"`
	// last l2 block height which is included in the batch
	End uint64 `json:"end"`

	LastSubmissionTime time.Time `json:"last_submission_time"`
	BatchFileSize      int64     `json:"batch_size"`
}

type BatchDataType uint8

const (
	BatchDataTypeHeader BatchDataType = iota
	BatchDataTypeChunk
)

func MarshalBatchDataHeader(
	start uint64,
	end uint64,
	checksums [][]byte,
) []byte {
	data := make([]byte, 1)
	data[0] = byte(BatchDataTypeHeader)
	data = binary.AppendUvarint(data, start)
	data = binary.AppendUvarint(data, end)
	data = binary.AppendUvarint(data, uint64(len(checksums)))
	for _, checksum := range checksums {
		data = append(data, checksum...)
	}
	return data
}

func UnmarshalBatchDataHeader(data []byte) (start uint64, end uint64, length uint64, checksums [][]byte, err error) {
	if len(data) < 25 {
		err = errors.New("invalid data length")
		return
	}
	start, _ = binary.Uvarint(data[1:9])
	end, _ = binary.Uvarint(data[9:17])
	length, _ = binary.Uvarint(data[17:25])
	checksums = make([][]byte, 0, length)

	if len(data)-25%32 != 0 || (uint64(len(data)-25)/32) != length {
		err = errors.New("invalid checksum data")
		return
	}

	for i := 25; i < len(data); i += 32 {
		checksums = append(checksums, data[i:i+32])
	}
	return
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
	data = binary.AppendUvarint(data, start)
	data = binary.AppendUvarint(data, end)
	data = binary.AppendUvarint(data, index)
	data = binary.AppendUvarint(data, length)
	data = append(data, chunkData...)
	return data
}

func UnmarshalBatchDataChunk(data []byte) (start uint64, end uint64, index uint64, length uint64, chunkData []byte, err error) {
	if len(data) < 33 {
		err = errors.New("invalid data length")
		return
	}
	start, _ = binary.Uvarint(data[1:9])
	end, _ = binary.Uvarint(data[9:17])
	index, _ = binary.Uvarint(data[17:25])
	length, _ = binary.Uvarint(data[25:33])
	chunkData = data[33:]
	return
}
