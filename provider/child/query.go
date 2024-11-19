package child

import (
	"context"
	"time"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"

	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/cosmos/cosmos-sdk/x/authz"

	"github.com/initia-labs/opinit-bots/node/rpcclient"
	"github.com/initia-labs/opinit-bots/types"
)

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

func (b BaseChild) QueryExecutors(ctx context.Context) ([]string, error) {
	req := &opchildtypes.QueryParamsRequest{}
	ctx, cancel := rpcclient.GetQueryContext(ctx, 0)
	defer cancel()

	res, err := b.opchildQueryClient.Params(ctx, req)
	if err != nil {
		return nil, err
	}
	return res.Params.BridgeExecutors, nil
}

func (b BaseChild) QueryGrantsRequest(ctx context.Context, granter, grantee, msgTypeUrl string) (*authz.QueryGrantsResponse, error) {
	req := &authz.QueryGrantsRequest{
		Granter:    granter,
		Grantee:    grantee,
		MsgTypeUrl: msgTypeUrl,
	}
	ctx, cancel := rpcclient.GetQueryContext(ctx, 0)
	defer cancel()

	authzClient := authz.NewQueryClient(b.node.GetRPCClient())
	res, err := authzClient.Grants(ctx, req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (b BaseChild) QueryGranteeGrants(ctx context.Context, grantee string) ([]*authz.GrantAuthorization, error) {
	req := &authz.QueryGranteeGrantsRequest{
		Grantee: grantee,
		Pagination: &query.PageRequest{
			Limit: 100,
		},
	}
	ctx, cancel := rpcclient.GetQueryContext(ctx, 0)
	defer cancel()

	authzClient := authz.NewQueryClient(b.node.GetRPCClient())

	ticker := time.NewTicker(types.PollingInterval(ctx))
	defer ticker.Stop()

	result := make([]*authz.GrantAuthorization, 0)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}

		res, err := authzClient.GranteeGrants(ctx, req)
		if err != nil {
			return nil, err
		}

		result = append(result, res.Grants...)
		if res.Pagination.NextKey == nil {
			break
		}
		req.Pagination.Key = res.Pagination.NextKey
	}

	return result, nil
}
