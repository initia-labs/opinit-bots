package child

import (
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (b BaseChild) GetMsgSetBridgeInfo(
	bridgeInfo opchildtypes.BridgeInfo,
) (sdk.Msg, error) {
	sender, err := b.node.MustGetBroadcaster().GetAddressString()
	if err != nil {
		return nil, err
	}

	msg := opchildtypes.NewMsgSetBridgeInfo(
		sender,
		bridgeInfo,
	)
	err = msg.Validate(b.node.AccountCodec())
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func (b BaseChild) GetMsgFinalizeTokenDeposit(
	from string,
	to string,
	coin sdk.Coin,
	l1Sequence uint64,
	blockHeight uint64,
	l1Denom string,
	data []byte,
) (sdk.Msg, error) {
	sender, err := b.node.MustGetBroadcaster().GetAddressString()
	if err != nil {
		return nil, err
	}

	msg := opchildtypes.NewMsgFinalizeTokenDeposit(
		sender,
		from,
		to,
		coin,
		l1Sequence,
		blockHeight,
		l1Denom,
		data,
	)
	err = msg.Validate(b.node.AccountCodec())
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func (b BaseChild) GetMsgUpdateOracle(
	height uint64,
	data []byte,
) (sdk.Msg, error) {
	sender, err := b.node.MustGetBroadcaster().GetAddressString()
	if err != nil {
		return nil, err
	}

	msg := opchildtypes.NewMsgUpdateOracle(
		sender,
		height,
		data,
	)
	err = msg.Validate(b.node.AccountCodec())
	if err != nil {
		return nil, err
	}
	return msg, nil
}
