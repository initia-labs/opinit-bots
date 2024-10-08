package child

import (
	"errors"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	"github.com/initia-labs/opinit-bots/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (b BaseChild) GetMsgFinalizeTokenDeposit(
	from string,
	to string,
	coin sdk.Coin,
	l1Sequence uint64,
	blockHeight int64,
	l1Denom string,
	data []byte,
) (sdk.Msg, error) {
	sender, err := b.GetAddressStr()
	if err != nil {
		if errors.Is(err, types.ErrKeyNotSet) {
			return nil, nil
		}
		return nil, err
	}

	msg := opchildtypes.NewMsgFinalizeTokenDeposit(
		sender,
		from,
		to,
		coin,
		l1Sequence,
		types.MustInt64ToUint64(blockHeight),
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
	height int64,
	data []byte,
) (sdk.Msg, error) {
	sender, err := b.GetAddressStr()
	if err != nil {
		if errors.Is(err, types.ErrKeyNotSet) {
			return nil, nil
		}
		return nil, err
	}

	msg := opchildtypes.NewMsgUpdateOracle(
		sender,
		types.MustInt64ToUint64(height),
		data,
	)
	err = msg.Validate(b.node.AccountCodec())
	if err != nil {
		return nil, err
	}
	return msg, nil
}
