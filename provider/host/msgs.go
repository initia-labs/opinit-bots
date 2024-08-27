package host

import (
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (b BaseHost) GetMsgProposeOutput(
	bridgeId uint64,
	outputIndex uint64,
	l2BlockNumber uint64,
	outputRoot []byte,
) (sdk.Msg, error) {
	sender, err := b.node.MustGetBroadcaster().GetAddressString()
	if err != nil {
		return nil, err
	}

	msg := ophosttypes.NewMsgProposeOutput(
		sender,
		bridgeId,
		outputIndex,
		l2BlockNumber,
		outputRoot,
	)
	err = msg.Validate(b.node.AccountCodec())
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func (b BaseHost) CreateBatchMsg(batchBytes []byte) (sdk.Msg, error) {
	submitter, err := b.node.MustGetBroadcaster().GetAddressString()
	if err != nil {
		return nil, err
	}

	msg := ophosttypes.NewMsgRecordBatch(
		submitter,
		uint64(b.bridgeId),
		batchBytes,
	)
	err = msg.Validate(b.node.AccountCodec())
	if err != nil {
		return nil, err
	}
	return msg, nil
}
