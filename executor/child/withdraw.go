package child

import (
	"errors"
	"strconv"
	"sync"
	"time"

	"cosmossdk.io/math"
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	"github.com/initia-labs/opinit-bots-go/types"

	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"

	comettpyes "github.com/cometbft/cometbft/types"
)

var (
	treeLoader = sync.Once{}
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
	ch.handleInitiateWithdrawal(l2Sequence, from, to, baseDenom, amount)
	return nil
}

func (ch *Child) handleInitiateWithdrawal(l2Sequence uint64, from string, to string, baseDenom string, amount uint64) {
	withdrawal := ophosttypes.GenerateWithdrawalHash(ch.BridgeId(), l2Sequence, from, to, baseDenom, amount)
	ch.blockWithdrawals = append(ch.blockWithdrawals, withdrawal[:])
}

func (ch *Child) prepareWithdrawals(blockHeight uint64) error {
	if ch.nextOutputTime.IsZero() || time.Now().After(ch.nextOutputTime) {
		output, err := ch.host.QueryLastOutput()
		if err != nil {
			return err
		}
		outputIndex := output.OutputIndex + 1
		if outputIndex != 1 {
			ch.nextOutputTime = output.OutputProposal.L1BlockTime.Add(ch.bridgeInfo.BridgeConfig.SubmissionInterval * 2 / 3)
		}
		// load once working tree
		treeLoader.Do(func() {
			err = ch.mk.LoadWorkingTree(outputIndex)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (ch *Child) handleBlockWithdrawals() ([]types.KV, error) {
	return ch.mk.InsertLeaves(ch.blockWithdrawals)
}

func (ch *Child) generateOutputRoot(version uint8, blockId []byte, blockHeader comettpyes.Header) ([]types.KV, error) {
	if time.Now().Before(ch.nextOutputTime) {
		// skip
		return nil, nil
	}

	kvs, storageRoot, err := ch.mk.FinishWorkingTree()
	if err != nil {
		return nil, err
	}
	sender, err := ch.host.GetAddressStr()
	if err != nil {
		return nil, err
	}

	outputRoot := ophosttypes.GenerateOutputRoot([]byte{version}, blockHeader.AppHash, storageRoot, blockId)
	msg := ophosttypes.NewMsgProposeOutput(
		sender,
		ch.BridgeId(),
		uint64(blockHeader.Height),
		outputRoot[:],
	)
	err = msg.Validate(ch.host.AccountCodec())
	if err != nil {
		return nil, err
	}
	ch.msgQueue = append(ch.msgQueue, msg)
	return kvs, nil
}
