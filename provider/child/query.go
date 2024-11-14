package child

import (
	"context"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/opinit-bots/node/rpcclient"
	"github.com/pkg/errors"
)

func (b BaseChild) GetAddress() (sdk.AccAddress, error) {
	broadcaster, err := b.node.GetBroadcaster()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get broadcaster")
	}
	return broadcaster.GetAddress(), nil
}

func (b BaseChild) GetAddressStr() (string, error) {
	broadcaster, err := b.node.GetBroadcaster()
	if err != nil {
		return "", errors.Wrap(err, "failed to get broadcaster")
	}
	return broadcaster.GetAddressString()
}

func (b BaseChild) QueryBridgeInfo(ctx context.Context) (opchildtypes.BridgeInfo, error) {
	req := &opchildtypes.QueryBridgeInfoRequest{}
	ctx, cancel := rpcclient.GetQueryContext(ctx, 0)
	defer cancel()

	res, err := b.opchildQueryClient.BridgeInfo(ctx, req)
	if err != nil {
		return opchildtypes.BridgeInfo{}, err
	}
	return res.BridgeInfo, nil
}

func (b BaseChild) QueryNextL1Sequence(ctx context.Context, height int64) (uint64, error) {
	req := &opchildtypes.QueryNextL1SequenceRequest{}
	ctx, cancel := rpcclient.GetQueryContext(ctx, height)
	defer cancel()

	res, err := b.opchildQueryClient.NextL1Sequence(ctx, req)
	if err != nil {
		return 0, err
	}
	return res.NextL1Sequence, nil
}

func (b BaseChild) QueryNextL2Sequence(ctx context.Context, height int64) (uint64, error) {
	req := &opchildtypes.QueryNextL2SequenceRequest{}
	ctx, cancel := rpcclient.GetQueryContext(ctx, height)
	defer cancel()

	res, err := b.opchildQueryClient.NextL2Sequence(ctx, req)
	if err != nil {
		return 0, err
	}
	return res.NextL2Sequence, nil
}
