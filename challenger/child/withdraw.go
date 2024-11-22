package child

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
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
	workingTree, err := merkle.GetWorkingTree(ch.DB(), types.MustInt64ToUint64(blockHeight-1))
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
	workingTree, err := ch.GetWorkingTree()
	if err != nil {
		return err
	}

	// initialize next output time
	if ch.nextOutputTime.IsZero() && workingTree.Index > 1 {
		output, err := ch.host.QuerySyncedOutput(ctx, ch.BridgeId(), workingTree.Index-1)
		if err != nil {
			// TODO: maybe not return error here and roll back
			return fmt.Errorf("output does not exist at index: %d", workingTree.Index-1)
		}
		ch.lastOutputTime = output.OutputProposal.L1BlockTime
	}

	output, err := ch.host.QuerySyncedOutput(ctx, ch.BridgeId(), workingTree.Index)
	if err != nil {
		if strings.Contains(err.Error(), "collections: not found") {
			// should check the existing output.
			return errors.Wrap(nodetypes.ErrIgnoreAndTryLater, fmt.Sprintf("output does not exist: %d", workingTree.Index))
		}
		return err
	} else {
		ch.nextOutputTime = output.OutputProposal.L1BlockTime
		ch.finalizingBlockHeight = types.MustUint64ToInt64(output.OutputProposal.L2BlockNumber)
	}
	return nil
}

func (ch *Child) handleTree(ctx types.Context, blockHeight int64, blockHeader cmtproto.Header) (storageRoot []byte, err error) {
	// panic if we passed the finalizing block height
	// this must not happened
	if ch.finalizingBlockHeight != 0 && ch.finalizingBlockHeight < blockHeight {
		panic(fmt.Errorf("INVARIANT failed; handleTree expect to finalize tree at block `%d` but we got block `%d`", ch.finalizingBlockHeight, blockHeight))
	}

	if ch.finalizingBlockHeight == blockHeight {
		finalizedTree, newNodes, treeRootHash, err := ch.Merkle().FinalizeWorkingTree(nil)
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

		workingTree, err := ch.GetWorkingTree()
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

		ch.finalizingBlockHeight = 0
		ch.lastOutputTime = blockHeader.Time
	}

	workingTree, err := ch.Merkle().GetWorkingTree()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get working tree")
	}

	err = merkle.SaveWorkingTree(ch.stage, workingTree)
	if err != nil {
		return nil, errors.Wrap(err, "failed to save working tree")
	}

	return storageRoot, nil
}

func (ch *Child) handleOutput(blockTime time.Time, blockHeight int64, version uint8, blockId []byte, outputIndex uint64, storageRoot []byte) error {
	outputRoot := ophosttypes.GenerateOutputRoot(version, storageRoot, blockId)
	output := challengertypes.NewOutput(blockHeight, outputIndex, outputRoot[:], blockTime)

	ch.eventQueue = append(ch.eventQueue, output)
	return nil
}
