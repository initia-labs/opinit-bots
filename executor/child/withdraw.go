package child

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"cosmossdk.io/math"
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	"github.com/initia-labs/opinit-bots-go/types"
	"go.uber.org/zap"

	dbtypes "github.com/initia-labs/opinit-bots-go/db/types"
	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"

	comettpyes "github.com/cometbft/cometbft/types"
)

func (ch *Child) initiateWithdrawalHandler(args nodetypes.EventHandlerArgs) error {
	var l2Sequence, amount uint64
	var from, to, baseDenom string
	var err error

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
				return errors.New("invalid amount")
			}
			amount = coinAmount.Uint64()
		}
	}
	return ch.handleInitiateWithdrawal(l2Sequence, from, to, baseDenom, amount)
}

func (ch *Child) handleInitiateWithdrawal(l2Sequence uint64, from string, to string, baseDenom string, amount uint64) error {
	withdrawal := ophosttypes.GenerateWithdrawalHash(ch.BridgeId(), l2Sequence, from, to, baseDenom, amount)
	err := ch.mk.InsertLeaf(withdrawal[:], false)
	if err != nil {
		return err
	}
	ch.logger.Info("initiate token withdrawal",
		zap.Uint64("l2_sequence", l2Sequence),
		zap.String("from", from),
		zap.String("to", to),
		zap.Uint64("amount", amount),
		zap.String("base_denom", baseDenom),
		zap.ByteString("withdrawal", withdrawal[:]),
	)
	return nil
}

func (ch *Child) prepareOutput(blockHeight uint64, blockTime time.Time) error {
	// we don't want query every block
	if ch.nextOutputTime.IsZero() || blockTime.After(ch.nextOutputTime) {
		output, err := ch.host.QueryLastOutput()
		if err != nil {
			return err
		}

		if ch.mk.GetWorkingTreeIndex() < output.OutputIndex+1 {
			// we are on sync; need to sync tree
			// store specific fianlizing height
			ch.finalizingBlockHeight = output.OutputProposal.L2BlockNumber
		}
		ch.nextOutputTime = output.OutputProposal.L1BlockTime.Add(ch.bridgeInfo.BridgeConfig.SubmissionInterval * 2 / 3)
	}
	return nil
}

func (ch *Child) prepareTree(blockHeight uint64) error {
	if blockHeight == 1 {
		ch.mk.SetNewWorkingTree(1, 1)
		return nil
	}

	err := ch.mk.LoadWorkingTree(blockHeight - 1)
	if err == dbtypes.ErrNotFound {
		// must not happend
		// TOOD: if user want to start from a specific height, we need to provide a way to do so
		panic(fmt.Errorf("working tree not found at height: %d, current: %d", blockHeight-1, blockHeight))
	} else if err != nil {
		return err
	}
	return nil
}

func (ch *Child) handleTree(blockHeight uint64, latestHeight uint64, blockId []byte, blockHeader comettpyes.Header) (kvs []types.KV, storageRoot []byte, err error) {
	// finalize working tree if we are on sync or block time is over next output time
	if ch.finalizingBlockHeight == blockHeight ||
		(ch.finalizingBlockHeight == 0 && blockHeight == latestHeight && blockHeader.Time.After(ch.nextOutputTime)) {
		// save blockId as extra data
		kvs, storageRoot, err = ch.mk.FinalizeWorkingTree(blockId)
		if err != nil {
			return nil, nil, err
		}

		// does not submit output since it already submitted
		if ch.finalizingBlockHeight == blockHeight {
			storageRoot = nil
		}
		ch.finalizingBlockHeight = 0
		ch.logger.Info("finalize tree", zap.Uint64("tree_index", ch.mk.GetWorkingTreeIndex()), zap.Uint64("height", blockHeight), zap.Uint64("num_leaves", ch.mk.GetWorkingTreeLeafCount()), zap.ByteString("root", storageRoot))
	}

	err = ch.mk.SaveWorkingTree(blockHeight)
	if err != nil {
		return nil, nil, err
	}
	return kvs, storageRoot, nil
}

func (ch *Child) handleOutput(blockHeight uint64, version uint8, blockId []byte, storageRoot []byte) error {
	sender, err := ch.host.GetAddressStr()
	if err != nil {
		return err
	}

	outputRoot := ophosttypes.GenerateOutputRoot(version, storageRoot, blockId)
	msg := ophosttypes.NewMsgProposeOutput(
		sender,
		ch.BridgeId(),
		blockHeight,
		outputRoot[:],
	)
	err = msg.Validate(ch.host.AccountCodec())
	if err != nil {
		return err
	}
	ch.msgQueue = append(ch.msgQueue, msg)
	return nil
}
