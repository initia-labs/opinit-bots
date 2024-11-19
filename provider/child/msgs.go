package child

import (
	"errors"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	"github.com/initia-labs/opinit-bots/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
)

func (b BaseChild) GetMsgFinalizeTokenDeposit(
	from string,
	to string,
	coin sdk.Coin,
	l1Sequence uint64,
	blockHeight int64,
	l1Denom string,
	data []byte,
) (sdk.Msg, string, error) {
	sender, err := b.BaseAccountAddressString()
	if err != nil {
		if errors.Is(err, types.ErrKeyNotSet) {
			return nil, "", nil
		}
		return nil, "", err
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
		return nil, "", err
	}
	return msg, sender, nil
}

func (b BaseChild) GetMsgUpdateOracle(
	height int64,
	data []byte,
) (sdk.Msg, string, error) {
	oracleAddress, err := b.OracleAccountAddress()
	if err != nil {
		if errors.Is(err, types.ErrKeyNotSet) {
			return nil, "", nil
		}
		return nil, "", err
	}
	oracleAddressString, err := b.OracleAccountAddressString()
	if err != nil {
		return nil, "", err
	}

	if b.oracleAccountGranter == "" {
		return nil, "", errors.New("oracle account granter is not set")
	}

	msg := opchildtypes.NewMsgUpdateOracle(
		b.oracleAccountGranter,
		types.MustInt64ToUint64(height),
		data,
	)
	err = msg.Validate(b.node.AccountCodec())
	if err != nil {
		return nil, "", err
	}

	authzMsgExec := authz.NewMsgExec(oracleAddress, []sdk.Msg{msg})
	return &authzMsgExec, oracleAddressString, nil
}
