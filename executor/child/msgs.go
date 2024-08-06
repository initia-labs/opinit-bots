package child

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
)

func (ch Child) GetMsgFinalizeTokenDeposit(
	from string,
	to string,
	coin sdk.Coin,
	l1Sequence uint64,
	blockHeight uint64,
	l1Denom string,
	data []byte,
) (sdk.Msg, error) {
	sender, err := ch.node.MustGetBroadcaster().GetAddressString()
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
	err = msg.Validate(ch.node.AccountCodec())
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func (ch Child) GetMsgUpdateOracle(
	height uint64,
	data []byte,
) (sdk.Msg, error) {
	sender, err := ch.node.MustGetBroadcaster().GetAddressString()
	if err != nil {
		return nil, err
	}

	msg := opchildtypes.NewMsgUpdateOracle(
		sender,
		height,
		data,
	)
	err = msg.Validate(ch.node.AccountCodec())
	if err != nil {
		return nil, err
	}
	return msg, nil
}
