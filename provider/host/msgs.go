package host

import (
	"errors"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	"github.com/initia-labs/opinit-bots/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (b BaseHost) GetMsgProposeOutput(
	bridgeId uint64,
	outputIndex uint64,
	l2BlockNumber int64,
	outputRoot []byte,
) (sdk.Msg, string, error) {
	sender, err := b.BaseAccountAddressString()
	if err != nil {
		if errors.Is(err, types.ErrKeyNotSet) {
			return nil, "", nil
		}
		return nil, "", err
	}

	msg := ophosttypes.NewMsgProposeOutput(
		sender,
		bridgeId,
		outputIndex,
		types.MustInt64ToUint64(l2BlockNumber),
		outputRoot,
	)
	err = msg.Validate(b.node.AccountCodec())
	if err != nil {
		return nil, "", err
	}
	return msg, sender, nil
}

func (b BaseHost) CreateBatchMsg(batchBytes []byte) (sdk.Msg, string, error) {
	submitter, err := b.BaseAccountAddressString()
	if err != nil {
		if errors.Is(err, types.ErrKeyNotSet) {
			return nil, "", nil
		}
		return nil, "", err
	}

	msg := ophosttypes.NewMsgRecordBatch(
		submitter,
		b.BridgeId(),
		batchBytes,
	)
	err = msg.Validate(b.node.AccountCodec())
	if err != nil {
		return nil, "", err
	}
	return msg, submitter, nil
}
