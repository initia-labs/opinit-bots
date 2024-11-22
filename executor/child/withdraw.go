package child

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	executortypes "github.com/initia-labs/opinit-bots/executor/types"
	"github.com/initia-labs/opinit-bots/merkle"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbtypes "github.com/initia-labs/opinit-bots/db/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	childprovider "github.com/initia-labs/opinit-bots/provider/child"
)

func (ch *Child) initiateWithdrawalHandler(ctx types.Context, args nodetypes.EventHandlerArgs) error {
	l2Sequence, amount, from, to, baseDenom, err := childprovider.ParseInitiateWithdrawal(args.EventAttributes)
	if err != nil {
		return errors.Wrap(err, "failed to parse initiate withdrawal event")
	}
	err = ch.handleInitiateWithdrawal(ctx, l2Sequence, from, to, baseDenom, amount)
	if err != nil {
		return errors.Wrap(err, "failed to handle initiate withdrawal")
	}
	return nil
}

func (ch *Child) handleInitiateWithdrawal(ctx types.Context, l2Sequence uint64, from string, to string, baseDenom string, amount uint64) error {
	withdrawalHash := ophosttypes.GenerateWithdrawalHash(ch.BridgeId(), l2Sequence, from, to, baseDenom, amount)
	data := executortypes.NewWithdrawalData(l2Sequence, from, to, amount, baseDenom, withdrawalHash[:])

	// store to database
	err := ch.SaveWithdrawal(l2Sequence, data)
	if err != nil {
		return errors.Wrap(err, "failed to save withdrawal data")
	}

	// generate merkle tree
	newNodes, err := ch.Merkle().InsertLeaf(withdrawalHash[:])
	if err != nil {
		return errors.Wrap(err, "failed to insert leaf to merkle tree")
	}
	err = merkle.SaveNodes(ch.stage, newNodes...)
	if err != nil {
		return errors.Wrap(err, "failed to save new tree nodes")
	}

	ctx.Logger().Info("initiate token withdrawal",
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
	workingTree, err := merkle.GetWorkingTree(ch.DB(), types.MustInt64ToUint64(blockHeight)-1)
	if err == dbtypes.ErrNotFound {
		if ch.InitializeTree(blockHeight) {
			return nil
		}
		// must not happened
		panic(fmt.Errorf("working tree not found at height: %d, current: %d", blockHeight-1, blockHeight))
	} else if err != nil {
		return errors.Wrap(err, "failed to get working tree")
	}

	err = ch.Merkle().LoadWorkingTree(workingTree)
	if err != nil {
		return errors.Wrap(err, "failed to load working tree")
	}

	return nil
}

func (ch *Child) prepareOutput(ctx context.Context) error {
	workingTree, err := ch.WorkingTree()
	if err != nil {
		return errors.Wrap(err, "failed to get working tree")
	}

	// initialize next output time
	if ch.nextOutputTime.IsZero() && workingTree.Index > 1 {
		output, err := ch.host.QueryOutput(ctx, ch.BridgeId(), workingTree.Index-1, 0)
		if err != nil {
			// TODO: maybe not return error here and roll back
			return fmt.Errorf("output does not exist at index: %d", workingTree.Index-1)
		}
		ch.lastOutputTime = output.OutputProposal.L1BlockTime
		ch.nextOutputTime = output.OutputProposal.L1BlockTime.Add(ch.BridgeInfo().BridgeConfig.SubmissionInterval * 2 / 3)
	}

	output, err := ch.host.QueryOutput(ctx, ch.BridgeId(), workingTree.Index, 0)
	if err != nil {
		if strings.Contains(err.Error(), "collections: not found") {
			return nil
		}
		return errors.Wrap(err, "failed to query output")
	} else {
		// we are syncing
		ch.finalizingBlockHeight = types.MustUint64ToInt64(output.OutputProposal.L2BlockNumber)
	}
	return nil
}

func (ch *Child) handleTree(ctx types.Context, blockHeight int64, latestHeight int64, blockId []byte, blockHeader cmtproto.Header) (storageRoot []byte, err error) {
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

		treeExtraData := executortypes.NewTreeExtraData(blockHeight, blockId)
		data, err := treeExtraData.Marshal()
		if err != nil {
			return nil, err
		}

		finalizedTree, newNodes, treeRootHash, err := ch.Merkle().FinalizeWorkingTree(data)
		if err != nil {
			return nil, errors.Wrap(err, "failed to finalize working tree")
		}

		if finalizedTree != nil {
			err = merkle.SaveFinalizedTree(ch.stage, *finalizedTree)
			if err != nil {
				return nil, errors.Wrap(err, "failed to save finalized tree")
			}
		}

		err = merkle.SaveNodes(ch.stage, newNodes...)
		if err != nil {
			return nil, errors.Wrap(err, "failed to save new nodes of finalized tree")
		}

		workingTree, err := ch.WorkingTree()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get working tree")
		}

		ctx.Logger().Info("finalize working tree",
			zap.Uint64("tree_index", workingTree.Index),
			zap.Int64("height", blockHeight),
			zap.Uint64("start_leaf_index", workingTree.StartLeafIndex),
			zap.Uint64("num_leaves", workingTree.LeafCount),
			zap.String("storage_root", base64.StdEncoding.EncodeToString(treeRootHash)),
		)

		// skip output submission when it is already submitted
		if ch.finalizingBlockHeight == blockHeight {
			storageRoot = nil
		}

		ch.finalizingBlockHeight = 0
		ch.lastOutputTime = blockHeader.Time
		ch.nextOutputTime = blockHeader.Time.Add(ch.BridgeInfo().BridgeConfig.SubmissionInterval * 2 / 3)
	}

	workingTree, err := ch.WorkingTree()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get working tree")
	}

	err = merkle.SaveWorkingTree(ch.stage, workingTree)
	if err != nil {
		return nil, errors.Wrap(err, "failed to save working tree")
	}

	return storageRoot, nil
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
		return errors.Wrap(err, "failed to get msg propose output")
	} else if msg != nil {
		ch.AppendMsgQueue(msg, sender)
	}
	return nil
}

// GetWithdrawal returns the withdrawal data for the given sequence from the database
func (ch *Child) GetWithdrawal(sequence uint64) (executortypes.WithdrawalData, error) {
	dataBytes, err := ch.DB().Get(executortypes.PrefixedWithdrawalSequence(sequence))
	if err != nil {
		return executortypes.WithdrawalData{}, errors.Wrap(err, "failed to get withdrawal data from db")
	}
	data := executortypes.WithdrawalData{}
	err = data.Unmarshal(dataBytes)
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
			return true, errors.Wrap(err, "failed to convert value to uint64")
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
		return nil, 0, 0, errors.Wrap(err, "failed to get last address index")
	}

	if descOrder {
		if offset > total || offset == 0 {
			offset = total
		}
		startKey := executortypes.PrefixedWithdrawalAddressIndex(address, offset)
		err = ch.DB().ReverseIterate(dbtypes.AppendSplitter(executortypes.PrefixedWithdrawalAddress(address)), startKey, fetchFn)
		if err != nil {
			return nil, 0, 0, errors.Wrap(err, "failed to iterate withdrawal address indices")
		}

		next = offset - count
	} else {
		if offset == 0 {
			offset = 1
		}
		startKey := executortypes.PrefixedWithdrawalAddressIndex(address, offset)
		err := ch.DB().Iterate(dbtypes.AppendSplitter(executortypes.PrefixedWithdrawalAddress(address)), startKey, fetchFn)
		if err != nil {
			return nil, 0, 0, err
		}

		next = offset + count
	}

	return sequences, next, total, nil
}

func (ch *Child) SaveWithdrawal(sequence uint64, data executortypes.WithdrawalData) error {
	addressIndex, err := ch.GetAddressIndex(data.To)
	if err != nil {
		return errors.Wrap(err, "failed to get address index")
	}
	ch.addressIndexMap[data.To] = addressIndex + 1

	withdrawalDataWithIndex := executortypes.NewWithdrawalDataWithIndex(data, ch.addressIndexMap[data.To])
	dataBytes, err := withdrawalDataWithIndex.Marshal()
	if err != nil {
		return err
	}

	err = ch.stage.Set(executortypes.PrefixedWithdrawalSequence(sequence), dataBytes)
	if err != nil {
		return errors.Wrap(err, "failed to save withdrawal data")
	}
	err = ch.stage.Set(executortypes.PrefixedWithdrawalAddressIndex(data.To, ch.addressIndexMap[data.To]), dbtypes.FromUint64(sequence))
	if err != nil {
		return errors.Wrap(err, "failed to save withdrawal address index")
	}
	return nil
}

func (ch *Child) GetAddressIndex(address string) (uint64, error) {
	if index, ok := ch.addressIndexMap[address]; !ok {
		lastIndex, err := ch.GetLastAddressIndex(address)
		if err != nil {
			return 0, errors.Wrap(err, "failed to get last address index")
		}
		return lastIndex, nil
	} else {
		return index, nil
	}
}

func (ch *Child) GetLastAddressIndex(address string) (lastIndex uint64, err error) {
	err = ch.DB().ReverseIterate(dbtypes.AppendSplitter(executortypes.PrefixedWithdrawalAddress(address)), nil, func(key, _ []byte) (bool, error) {
		lastIndex = dbtypes.ToUint64Key(key[len(key)-8:])
		return true, nil
	})
	return lastIndex, err
}

func (ch *Child) DeleteFutureWithdrawals(fromSequence uint64) error {
	return ch.DB().Iterate(dbtypes.AppendSplitter(executortypes.WithdrawalSequencePrefix), nil, func(key, value []byte) (bool, error) {
		sequence := dbtypes.ToUint64Key(key[len(key)-8:])
		if sequence < fromSequence {
			return false, nil
		}

		data := executortypes.WithdrawalDataWithIndex{}
		err := data.Unmarshal(value)
		if err != nil {
			return true, err
		}
		err = ch.DB().Delete(executortypes.PrefixedWithdrawalAddressIndex(data.Withdrawal.To, data.Index))
		if err != nil {
			return true, errors.Wrap(err, "failed to delete withdrawal address index")
		}
		err = ch.DB().Delete(key)
		if err != nil {
			return true, errors.Wrap(err, "failed to delete withdrawal data")
		}

		return false, nil
	})
}
