package child

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"cosmossdk.io/math"
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	executortypes "github.com/initia-labs/opinit-bots-go/executor/types"
	"github.com/initia-labs/opinit-bots-go/types"
	"go.uber.org/zap"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbtypes "github.com/initia-labs/opinit-bots-go/db/types"
	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"
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
	withdrawalHash := ophosttypes.GenerateWithdrawalHash(ch.BridgeId(), l2Sequence, from, to, baseDenom, amount)
	data := executortypes.WithdrawalData{
		Sequence:       l2Sequence,
		From:           from,
		To:             to,
		Amount:         amount,
		BaseDenom:      baseDenom,
		WithdrawalHash: withdrawalHash[:],
	}
	err := ch.SetWithdrawal(l2Sequence, data)
	if err != nil {
		return err
	}
	err = ch.mk.InsertLeaf(withdrawalHash[:], false)
	if err != nil {
		return err
	}
	ch.logger.Info("initiate token withdrawal",
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

func (ch *Child) prepareOutput() error {
	workingOutputIndex := ch.mk.GetWorkingTreeIndex()
	// initialize next output time
	if ch.nextOutputTime.IsZero() && workingOutputIndex > 1 {
		output, err := ch.host.QueryOutput(workingOutputIndex - 1)
		if err != nil {
			// TODO: maybe not panic here and roll back
			panic(fmt.Errorf("output does not exist at index: %d", workingOutputIndex-1))
		}
		ch.nextOutputTime = output.OutputProposal.L1BlockTime.Add(ch.bridgeInfo.BridgeConfig.SubmissionInterval * 2 / 3)
	}

	output, err := ch.host.QueryOutput(ch.mk.GetWorkingTreeIndex())
	if err != nil {
		if strings.Contains(err.Error(), "collections: not found") {
			return nil
		}
		return err
	} else {
		// we are syncing
		ch.finalizingBlockHeight = output.OutputProposal.L2BlockNumber
	}
	return nil
}

func (ch *Child) handleTree(blockHeight uint64, latestHeight uint64, blockId []byte, blockHeader cmtproto.Header) (kvs []types.KV, storageRoot []byte, err error) {
	// finalize working tree if we are syncing or block time is over next output time
	if ch.finalizingBlockHeight == blockHeight ||
		(ch.finalizingBlockHeight == 0 && blockHeight == latestHeight && blockHeader.Time.After(ch.nextOutputTime)) {
		extraData := executortypes.TreeExtraData{
			BlockNumber: blockHeight,
			BlockHash:   blockId,
		}
		data, err := json.Marshal(extraData)
		if err != nil {
			return nil, nil, err
		}
		kvs, storageRoot, err = ch.mk.FinalizeWorkingTree(data)
		if err != nil {
			return nil, nil, err
		}
		ch.logger.Info("finalize tree", zap.Uint64("tree_index", ch.mk.GetWorkingTreeIndex()), zap.Uint64("height", blockHeight), zap.Uint64("num_leaves", ch.mk.GetWorkingTreeLeafCount()), zap.String("storage_root", base64.StdEncoding.EncodeToString(storageRoot)))

		// does not submit output since it already submitted
		if ch.finalizingBlockHeight == blockHeight {
			storageRoot = nil
		}
		ch.finalizingBlockHeight = 0
		ch.nextOutputTime = blockHeader.Time.Add(ch.bridgeInfo.BridgeConfig.SubmissionInterval * 2 / 3)
	}
	err = ch.mk.SaveWorkingTree(blockHeight)
	if err != nil {
		return nil, nil, err
	}
	return kvs, storageRoot, nil
}

func (ch *Child) handleOutput(blockHeight uint64, version uint8, blockId []byte, outputIndex uint64, storageRoot []byte) error {
	sender, err := ch.host.GetAddressStr()
	if err != nil {
		return err
	}

	outputRoot := ophosttypes.GenerateOutputRoot(version, storageRoot, blockId)
	msg := ophosttypes.NewMsgProposeOutput(
		sender,
		ch.BridgeId(),
		outputIndex,
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

func (ch *Child) GetWithdrawal(sequence uint64) (executortypes.WithdrawalData, error) {
	dataBytes, err := ch.db.Get(executortypes.PrefixedWithdrawalKey(sequence))
	if err != nil {
		return executortypes.WithdrawalData{}, err
	}
	var data executortypes.WithdrawalData
	err = json.Unmarshal(dataBytes, &data)
	return data, err
}

func (ch *Child) SetWithdrawal(sequence uint64, data executortypes.WithdrawalData) error {
	dataBytes, err := json.Marshal(&data)
	if err != nil {
		return err
	}

	return ch.db.Set(executortypes.PrefixedWithdrawalKey(sequence), dataBytes)
}
