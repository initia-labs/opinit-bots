package host

import (
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (h Host) GetMsgProposeOutput(
	bridgeId uint64,
	outputIndex uint64,
	l2BlockNumber uint64,
	outputRoot []byte,
) (sdk.Msg, error) {
	sender, err := h.Node().MustGetBroadcaster().GetAddressString()
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
	err = msg.Validate(h.Node().AccountCodec())
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func (h Host) CreateBatchMsg(batchBytes []byte) (sdk.Msg, error) {
	submitter, err := h.Node().MustGetBroadcaster().GetAddressString()
	if err != nil {
		return nil, err
	}

	msg := ophosttypes.NewMsgRecordBatch(
		submitter,
		uint64(h.BridgeId()),
		batchBytes,
	)
	err = msg.Validate(h.Node().AccountCodec())
	if err != nil {
		return nil, err
	}
	return msg, nil
}
