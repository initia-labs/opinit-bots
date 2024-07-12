package child

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
)

func (ch Child) GetAddress() sdk.AccAddress {
	return ch.node.GetAddress()
}

func (ch Child) GetAddressStr() (string, error) {
	return ch.ac.BytesToString(ch.node.GetAddress())
}

func (ch Child) QueryBridgeInfo() (opchildtypes.BridgeInfo, error) {
	req := &opchildtypes.QueryBridgeInfoRequest{}
	res, err := ch.opchildQueryClient.BridgeInfo(context.Background(), req)
	if err != nil {
		return opchildtypes.BridgeInfo{}, err
	}
	return res.BridgeInfo, nil
}
