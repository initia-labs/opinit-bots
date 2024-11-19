package child

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	executortypes "github.com/initia-labs/opinit-bots/executor/types"
	"github.com/initia-labs/opinit-bots/types"
	"go.uber.org/zap"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbtypes "github.com/initia-labs/opinit-bots/db/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	childprovider "github.com/initia-labs/opinit-bots/provider/child"
)

func (ch *Child) initiateWithdrawalHandler(_ context.Context, args nodetypes.EventHandlerArgs) error {
	l2Sequence, amount, from, to, baseDenom, err := childprovider.ParseInitiateWithdrawal(args.EventAttributes)
	if err != nil {
		return err
	}
	return ch.handleInitiateWithdrawal(l2Sequence, from, to, baseDenom, amount)
}

func (ch *Child) handleInitiateWithdrawal(l2Sequence uint64, from string, to string, baseDenom string, amount uint64) error {
	withdrawalHash := ophosttypes.GenerateWithdrawalHash(ch.BridgeId(), l2Sequence, from, to, baseDenom, amount)
	data := executortypes.WithdrawalData{
		Sequence:       l2Sequence,
		From:           from,
		To:             to,
		Amount:         amount,
		BaseDenom:      baseDenom,
		WithdrawalHash: withdrawalHash[:],
	}

	// store to database
	kvs, err := ch.WithdrawalToRawKVs(l2Sequence, data)
	if err != nil {
		return err
	}
	ch.batchKVs = append(ch.batchKVs, kvs...)

	// generate merkle tree
	err = ch.Merkle().InsertLeaf(withdrawalHash[:])
	if err != nil {
		return err
	}

	ch.Logger().Info("initiate token withdrawal",
		zap.Uint64("l2_sequence", l2Sequence),
		zap.String("from", from),
		zap.String("to", to),
		zap.Uint64("amount", amount),
		zap.String("base_denom", baseDenom),
		zap.String("withdrawal", base64.StdEncoding.EncodeToString(withdrawalHash[:])),
	)

	return nil
}

func (ch *Child) prepareTree(blockHeight int64) error {
	if ch.InitializeTree(blockHeight) {
		return nil
	}

	err := ch.Merkle().LoadWorkingTree(types.MustInt64ToUint64(blockHeight) - 1)
	if err == dbtypes.ErrNotFound {
		// must not happened
		panic(fmt.Errorf("working tree not found at height: %d, current: %d", blockHeight-1, blockHeight))
	} else if err != nil {
		return err
	}

	return nil
}

func (ch *Child) prepareOutput(ctx context.Context) error {
	workingTreeIndex, err := ch.GetWorkingTreeIndex()
	if err != nil {
		return err
	}

	// initialize next output time
	if ch.nextOutputTime.IsZero() && workingTreeIndex > 1 {
		output, err := ch.host.QueryOutput(ctx, ch.BridgeId(), workingTreeIndex-1, 0)
		if err != nil {
			// TODO: maybe not return error here and roll back
			return fmt.Errorf("output does not exist at index: %d", workingTreeIndex-1)
		}
		ch.lastOutputTime = output.OutputProposal.L1BlockTime
		ch.nextOutputTime = output.OutputProposal.L1BlockTime.Add(ch.BridgeInfo().BridgeConfig.SubmissionInterval * 2 / 3)
	}

	output, err := ch.host.QueryOutput(ctx, ch.BridgeId(), workingTreeIndex, 0)
	if err != nil {
		if strings.Contains(err.Error(), "collections: not found") {
			return nil
		}
		return err
	} else {
		// we are syncing
		ch.finalizingBlockHeight = types.MustUint64ToInt64(output.OutputProposal.L2BlockNumber)
	}
	return nil
}

func (ch *Child) handleTree(blockHeight int64, latestHeight int64, blockId []byte, blockHeader cmtproto.Header) (kvs []types.RawKV, storageRoot []byte, err error) {
	// panic if we are syncing and passed the finalizing block height
	// this must not happened
	if ch.finalizingBlockHeight != 0 && ch.finalizingBlockHeight < blockHeight {
		panic(fmt.Errorf("INVARIANT failed; handleTree expect to finalize tree at block `%d` but we got block `%d`", blockHeight-1, blockHeight))
	}

	// finalize working tree if we are fully synced or block time is over next output time
	if ch.finalizingBlockHeight == blockHeight ||
		(ch.finalizingBlockHeight == 0 &&
			blockHeight == latestHeight &&
			blockHeader.Time.After(ch.nextOutputTime)) {

		data, err := json.Marshal(executortypes.TreeExtraData{
			BlockNumber: blockHeight,
			BlockHash:   blockId,
		})
		if err != nil {
			return nil, nil, err
		}

		kvs, storageRoot, err = ch.Merkle().FinalizeWorkingTree(data)
		if err != nil {
			return nil, nil, err
		}

		workingTreeIndex, err := ch.GetWorkingTreeIndex()
		if err != nil {
			return nil, nil, err
		}

		workingTreeLeafCount, err := ch.GetWorkingTreeLeafCount()
		if err != nil {
			return nil, nil, err
		}

		startLeafIndex, err := ch.GetStartLeafIndex()
		if err != nil {
			return nil, nil, err
		}

		ch.Logger().Info("finalize working tree",
			zap.Uint64("tree_index", workingTreeIndex),
			zap.Int64("height", blockHeight),
			zap.Uint64("start_leaf_index", startLeafIndex),
			zap.Uint64("num_leaves", workingTreeLeafCount),
			zap.String("storage_root", base64.StdEncoding.EncodeToString(storageRoot)),
		)

		// skip output submission when it is already submitted
		if ch.finalizingBlockHeight == blockHeight {
			storageRoot = nil
		}

		ch.finalizingBlockHeight = 0
		ch.lastOutputTime = blockHeader.Time
		ch.nextOutputTime = blockHeader.Time.Add(ch.BridgeInfo().BridgeConfig.SubmissionInterval * 2 / 3)
	}

	version := types.MustInt64ToUint64(blockHeight)
	err = ch.Merkle().SaveWorkingTree(version)
	if err != nil {
		return nil, nil, err
	}

	return kvs, storageRoot, nil
}

func (ch *Child) handleOutput(blockHeight int64, version uint8, blockId []byte, outputIndex uint64, storageRoot []byte) error {
	outputRoot := ophosttypes.GenerateOutputRoot(version, storageRoot, blockId)
	msg, sender, err := ch.host.GetMsgProposeOutput(
		ch.BridgeId(),
		outputIndex,
		blockHeight,
		outputRoot[:],
	)
	if err != nil {
		return err
	} else if msg != nil {
		ch.AppendMsgQueue(msg, sender)
	}
	return nil
}

// GetWithdrawal returns the withdrawal data for the given sequence from the database
func (ch *Child) GetWithdrawal(sequence uint64) (executortypes.WithdrawalData, error) {
	dataBytes, err := ch.DB().Get(executortypes.PrefixedWithdrawalKey(sequence))
	if err != nil {
		return executortypes.WithdrawalData{}, err
	}
	var data executortypes.WithdrawalData
	err = json.Unmarshal(dataBytes, &data)
	return data, err
}

func (ch *Child) GetSequencesByAddress(address string, offset uint64, limit uint64, descOrder bool) (sequences []uint64, next, total uint64, err error) {
	if limit == 0 {
		return nil, 0, 0, nil
	}

	count := uint64(0)
	fetchFn := func(key, value []byte) (bool, error) {
		sequence, err := dbtypes.ToUint64(value)
		if err != nil {
			return true, err
		}
		sequences = append(sequences, sequence)
		count++
		if count >= limit {
			return true, nil
		}
		return false, nil
	}
	total, err = ch.GetLastAddressIndex(address)
	if err != nil {
		return nil, 0, 0, err
	}

	if descOrder {
		if offset > total || offset == 0 {
			offset = total
		}
		startKey := executortypes.PrefixedWithdrawalKeyAddressIndex(address, offset)
		err = ch.DB().PrefixedReverseIterate(executortypes.PrefixedWithdrawalKeyAddress(address), startKey, fetchFn)
		if err != nil {
			return nil, 0, 0, err
		}

		next = offset - count
	} else {
		if offset == 0 {
			offset = 1
		}
		startKey := executortypes.PrefixedWithdrawalKeyAddressIndex(address, offset)
		err := ch.DB().PrefixedIterate(executortypes.PrefixedWithdrawalKeyAddress(address), startKey, fetchFn)
		if err != nil {
			return nil, 0, 0, err
		}

		next = offset + count
	}

	return sequences, next, total, nil
}

// SetWithdrawal store the withdrawal data for the given sequence to the database
func (ch *Child) WithdrawalToRawKVs(sequence uint64, data executortypes.WithdrawalData) ([]types.RawKV, error) {
	dataBytes, err := json.Marshal(&data)
	if err != nil {
		return nil, err
	}

	kvs := make([]types.RawKV, 0)
	kvs = append(kvs, types.RawKV{
		Key:   ch.DB().PrefixedKey(executortypes.PrefixedWithdrawalKey(sequence)),
		Value: dataBytes,
	})

	addressIndex, err := ch.GetAddressIndex(data.To)
	if err != nil {
		return nil, err
	}
	ch.addressIndexMap[data.To] = addressIndex + 1
	kvs = append(kvs, types.RawKV{
		Key:   ch.DB().PrefixedKey(executortypes.PrefixedWithdrawalKeyAddressIndex(data.To, ch.addressIndexMap[data.To])),
		Value: dbtypes.FromUint64(sequence),
	})
	return kvs, nil
}

func (ch *Child) GetAddressIndex(address string) (uint64, error) {
	if index, ok := ch.addressIndexMap[address]; !ok {
		lastIndex, err := ch.GetLastAddressIndex(address)
		if err != nil {
			return 0, err
		}
		return lastIndex, nil
	} else {
		return index, nil
	}
}

func (ch *Child) GetLastAddressIndex(address string) (lastIndex uint64, err error) {
	err = ch.DB().PrefixedReverseIterate(executortypes.PrefixedWithdrawalKeyAddress(address), nil, func(key, _ []byte) (bool, error) {
		lastIndex = dbtypes.ToUint64Key(key[len(key)-8:])
		return true, nil
	})
	return lastIndex, err
}

func (ch *Child) DeleteFutureWithdrawals(fromSequence uint64) error {
	return ch.DB().PrefixedIterate(executortypes.WithdrawalKey, nil, func(key, _ []byte) (bool, error) {
		if len(key) != len(executortypes.WithdrawalKey)+1+8 {
			return false, nil
		}
		sequence := dbtypes.ToUint64Key(key[len(key)-8:])
		if sequence >= fromSequence {
			err := ch.DB().Delete(key)
			if err != nil {
				return true, err
			}
		}
		return false, nil
	})
}
