package types

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/pkg/errors"

	"github.com/cosmos/cosmos-sdk/codec"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type DANode interface {
	DB() types.DB
	Codec() codec.Codec

	Start(types.Context)
	HasBroadcaster() bool
	BroadcastProcessedMsgs(...btypes.ProcessedMsgs)

	CreateBatchMsg([]byte) (sdk.Msg, string, error)

	GetNodeStatus() (nodetypes.Status, error)

	LenProcessedBatchMsgs() (int, error)
	LenPendingBatchTxs() (int, error)

	LastUpdatedBatchTime() time.Time
}

var LocalBatchInfoKey = []byte("local_batch_info")

type LocalBatchInfo struct {
	// start l2 block height which is included in the batch
	Start int64 `json:"start"`
	// last l2 block height which is included in the batch
	End int64 `json:"end"`
	// last submission time of the batch
	LastSubmissionTime time.Time `json:"last_submission_time"`
	// batch file size
	BatchSize int64 `json:"batch_size"`
}

func (l LocalBatchInfo) Key() []byte {
	return LocalBatchInfoKey
}

func (l LocalBatchInfo) Value() ([]byte, error) {
	bz, err := l.Marshal()
	if err != nil {
		return nil, err
	}
	return bz, nil
}

func (l LocalBatchInfo) Marshal() ([]byte, error) {
	bz, err := json.Marshal(l)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal local batch info")
	}
	return bz, nil
}

func (l *LocalBatchInfo) Unmarshal(bz []byte) error {
	if err := json.Unmarshal(bz, l); err != nil {
		return errors.Wrap(err, "failed to unmarshal local batch info")
	}
	return nil
}

type BatchDataType uint8

const (
	BatchDataTypeHeader BatchDataType = iota
	BatchDataTypeChunk
	BatchDataTypeGenesis
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

type BatchDataGenesis struct {
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

func MarshalBatchDataGenesis(
	index uint64,
	length uint64,
	chunkData []byte,
) []byte {
	data := make([]byte, 1)
	data[0] = byte(BatchDataTypeGenesis)
	data = binary.BigEndian.AppendUint64(data, index)
	data = binary.BigEndian.AppendUint64(data, length)
	data = append(data, chunkData...)
	return data
}

func UnmarshalBatchDataGenesis(data []byte) (BatchDataGenesis, error) {
	if len(data) < 17 {
		err := fmt.Errorf("invalid data length: %d, expected > 17", len(data))
		return BatchDataGenesis{}, err
	}
	index := binary.BigEndian.Uint64(data[1:9])
	length := binary.BigEndian.Uint64(data[9:17])
	chunkData := data[17:]

	return BatchDataGenesis{
		Index:     index,
		Length:    length,
		ChunkData: chunkData,
	}, nil
}
