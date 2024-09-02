package child

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"cosmossdk.io/math"
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/pkg/errors"
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

	for _, attr := range args.EventAttributes {
		switch attr.Key {
		case opchildtypes.AttributeKeyL2Sequence:
			l2Sequence, err = strconv.ParseUint(attr.Value, 10, 64)
			if err != nil {
				return err
			}
		case opchildtypes.AttributeKeyFrom:
			from = attr.Value
		case opchildtypes.AttributeKeyTo:
			to = attr.Value
		case opchildtypes.AttributeKeyBaseDenom:
			baseDenom = attr.Value
		case opchildtypes.AttributeKeyAmount:
			coinAmount, ok := math.NewIntFromString(attr.Value)
			if !ok {
				return fmt.Errorf("invalid amount %s", attr.Value)
			}

			amount = coinAmount.Uint64()
		}
	}

	return ch.handleInitiateWithdrawal(l2Sequence, from, to, baseDenom, amount)
}

func (ch *Child) handleInitiateWithdrawal(l2Sequence uint64, from string, to string, baseDenom string, amount uint64) error {
	withdrawalHash := ophosttypes.GenerateWithdrawalHash(ch.BridgeId(), l2Sequence, from, to, baseDenom, amount)
	// generate merkle tree
	err := ch.Merkle().InsertLeaf(withdrawalHash[:])
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

func (ch *Child) prepareTree(blockHeight uint64) error {
	if ch.InitializeTree(blockHeight) {
		return nil
	}

	err := ch.Merkle().LoadWorkingTree(blockHeight - 1)
	if err == dbtypes.ErrNotFound {
		// must not happened
		panic(fmt.Errorf("working tree not found at height: %d, current: %d", blockHeight-1, blockHeight))
	} else if err != nil {
		return err
	}
	return nil
}

func (ch *Child) prepareOutput(ctx context.Context) error {
	workingOutputIndex := ch.Merkle().GetWorkingTreeIndex()

	// initialize next output time
	if ch.nextOutputTime.IsZero() && workingOutputIndex > 1 {
		output, err := ch.host.QuerySyncedOutput(ctx, ch.BridgeId(), workingOutputIndex-1)
		if err != nil {
			// TODO: maybe not return error here and roll back
			return fmt.Errorf("output does not exist at index: %d", workingOutputIndex-1)
		}
		ch.lastOutputTime = output.OutputProposal.L1BlockTime
	}

	output, err := ch.host.QuerySyncedOutput(ctx, ch.BridgeId(), ch.Merkle().GetWorkingTreeIndex())
	if err != nil {
		if strings.Contains(err.Error(), "collections: not found") {
			// should check the existing output.
			return errors.Wrap(nodetypes.ErrIgnoreAndTryLater, fmt.Sprintf("output does not exist: %d", ch.Merkle().GetWorkingTreeIndex()))
		}
		return err
	} else {
		ch.nextOutputTime = output.OutputProposal.L1BlockTime
		ch.finalizingBlockHeight = output.OutputProposal.L2BlockNumber
	}
	return nil
}

func (ch *Child) handleTree(blockHeight uint64, blockHeader cmtproto.Header) (kvs []types.RawKV, storageRoot []byte, err error) {
	// panic if we passed the finalizing block height
	// this must not happened
	if ch.finalizingBlockHeight != 0 && ch.finalizingBlockHeight < blockHeight {
		panic(fmt.Errorf("INVARIANT failed; handleTree expect to finalize tree at block `%d` but we got block `%d`", blockHeight-1, blockHeight))
	}

	if ch.finalizingBlockHeight == blockHeight {
		kvs, storageRoot, err = ch.Merkle().FinalizeWorkingTree(nil)
		if err != nil {
			return nil, nil, err
		}

		ch.Logger().Info("finalize working tree",
			zap.Uint64("tree_index", ch.Merkle().GetWorkingTreeIndex()),
			zap.Uint64("height", blockHeight),
			zap.Uint64("num_leaves", ch.Merkle().GetWorkingTreeLeafCount()),
			zap.String("storage_root", base64.StdEncoding.EncodeToString(storageRoot)),
		)

		ch.finalizingBlockHeight = 0
		ch.lastOutputTime = blockHeader.Time
	}

	err = ch.Merkle().SaveWorkingTree(blockHeight)
	if err != nil {
		return nil, nil, err
	}

	return kvs, storageRoot, nil
}

func (ch *Child) handleOutput(blockTime time.Time, blockHeight uint64, version uint8, blockId []byte, outputIndex uint64, storageRoot []byte) error {
	outputRoot := ophosttypes.GenerateOutputRoot(version, storageRoot, blockId)
	output := challengertypes.NewOutput(blockHeight, outputIndex, outputRoot[:], blockTime)

	ch.eventQueue = append(ch.eventQueue, output)
	return nil
}
