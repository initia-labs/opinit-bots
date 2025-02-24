package child

import (
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	"github.com/initia-labs/opinit-bots/types"

	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"

	"github.com/pkg/errors"
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
		return nil, "", errors.Wrap(err, "failed to get address")
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
		return nil, "", errors.Wrap(err, "failed to validate msg")
	}
	return msg, sender, nil
}

func (b BaseChild) GetMsgUpdateOracle(
	height int64,
	data []byte,
) (sdk.Msg, string, error) {
	oracleAddressString, err := b.OracleAccountAddressString()
	if err != nil {
		if errors.Is(err, types.ErrKeyNotSet) {
			return nil, "", nil
		}
		return nil, "", errors.Wrap(err, "failed to get address")
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
		return nil, "", errors.Wrap(err, "failed to validate msg")
	}

	authzMsg, err := CreateAuthzMsg(oracleAddressString, msg)
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to create authz msg")
	}
	return authzMsg, oracleAddressString, nil
}

func (b BaseChild) GetMsgSetBridgeInfo(
	bridgeId uint64,
	bridgeConfig ophosttypes.BridgeConfig,
) (sdk.Msg, string, error) {
	sender, err := b.BaseAccountAddressString()
	if err != nil {
		if errors.Is(err, types.ErrKeyNotSet) {
			return nil, "", nil
		}
		return nil, "", errors.Wrap(err, "failed to get address")
	}

	newBridgeInfo := b.BridgeInfo()
	newBridgeInfo.BridgeConfig = bridgeConfig

	msg := opchildtypes.NewMsgSetBridgeInfo(
		sender,
		newBridgeInfo,
	)
	err = msg.Validate(b.node.AccountCodec())
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to validate msg")
	}
	return msg, sender, nil
}

func CreateAuthzMsg(grantee string, msg sdk.Msg) (sdk.Msg, error) {
	msgsAny := make([]*cdctypes.Any, 1)
	any, err := cdctypes.NewAnyWithValue(msg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create any")
	}
	msgsAny[0] = any

	return &authz.MsgExec{
		Grantee: grantee,
		Msgs:    msgsAny,
	}, err
}
