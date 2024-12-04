package batchsubmitter

import (
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	"github.com/initia-labs/opinit-bots/types"
)

// UpdateBatchInfo appends the batch info with the given chain, submitter, output index, and l2 block number
func (bs *BatchSubmitter) UpdateBatchInfo(chain string, submitter string, outputIndex uint64, l2BlockNumber int64) {
	bs.batchInfoMu.Lock()
	defer bs.batchInfoMu.Unlock()

	if len(bs.batchInfos) == 0 {
		panic("batch info must be set before starting the batch submitter")
	}

	// check if the batch info is already updated
	if types.MustUint64ToInt64(bs.batchInfos[len(bs.batchInfos)-1].Output.L2BlockNumber) >= l2BlockNumber {
		return
	}

	bs.batchInfos = append(bs.batchInfos, ophosttypes.BatchInfoWithOutput{
		BatchInfo: ophosttypes.BatchInfo{
			ChainType: ophosttypes.BatchInfo_ChainType(ophosttypes.BatchInfo_ChainType_value["CHAIN_TYPE_"+chain]),
			Submitter: submitter,
		},
		Output: ophosttypes.Output{
			L2BlockNumber: types.MustInt64ToUint64(l2BlockNumber),
		},
	})
}

// BatchInfo returns the current batch info
func (bs *BatchSubmitter) BatchInfo() *ophosttypes.BatchInfoWithOutput {
	bs.batchInfoMu.Lock()
	defer bs.batchInfoMu.Unlock()

	return &bs.batchInfos[0]
}

// NextBatchInfo returns the next batch info in the queue
func (bs *BatchSubmitter) NextBatchInfo() *ophosttypes.BatchInfoWithOutput {
	bs.batchInfoMu.Lock()
	defer bs.batchInfoMu.Unlock()
	if len(bs.batchInfos) == 1 {
		return nil
	}
	return &bs.batchInfos[1]
}

// DequeueBatchInfo removes the first batch info from the queue
func (bs *BatchSubmitter) DequeueBatchInfo() {
	bs.batchInfoMu.Lock()
	defer bs.batchInfoMu.Unlock()

	bs.batchInfos = bs.batchInfos[1:]
}
