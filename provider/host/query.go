package host

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	query "github.com/cosmos/cosmos-sdk/types/query"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"

	"github.com/initia-labs/opinit-bots/node/rpcclient"
	"github.com/initia-labs/opinit-bots/types"
)

func (b BaseHost) GetAddress() sdk.AccAddress {
	return b.node.MustGetBroadcaster().GetAddress()
}

func (b BaseHost) GetAddressStr() (string, error) {
	return b.node.MustGetBroadcaster().GetAddressString()
}

func (b BaseHost) QueryLastOutput(ctx context.Context, bridgeId uint64) (*ophosttypes.QueryOutputProposalResponse, error) {
	req := &ophosttypes.QueryOutputProposalsRequest{
		BridgeId: bridgeId,
		Pagination: &query.PageRequest{
			Limit:   1,
			Reverse: true,
		},
	}
	ctx, cancel := rpcclient.GetQueryContext(ctx, 0)
	defer cancel()

	res, err := b.ophostQueryClient.OutputProposals(ctx, req)
	if err != nil {
		return nil, err
	}
	if res.OutputProposals == nil || len(res.OutputProposals) == 0 {
		return nil, nil
	}
	return &res.OutputProposals[0], nil
}

func (b BaseHost) QueryOutput(ctx context.Context, bridgeId uint64, outputIndex uint64) (*ophosttypes.QueryOutputProposalResponse, error) {
	req := &ophosttypes.QueryOutputProposalRequest{
		BridgeId:    bridgeId,
		OutputIndex: outputIndex,
	}
	ctx, cancel := rpcclient.GetQueryContext(ctx, 0)
	defer cancel()

	return b.ophostQueryClient.OutputProposal(ctx, req)
}

// QueryOutputByL2BlockNumber queries the last output proposal before the given L2 block number
func (b BaseHost) QueryOutputByL2BlockNumber(ctx context.Context, bridgeId uint64, l2BlockNumber uint64) (*ophosttypes.QueryOutputProposalResponse, error) {
	start, err := b.QueryOutput(ctx, bridgeId, 1)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, nil
		}
		return nil, err
	}
	end, err := b.QueryLastOutput(ctx, bridgeId)
	if err != nil {
		return nil, err
	} else if end == nil {
		return nil, nil
	}

	for {
		if start.OutputProposal.L2BlockNumber >= l2BlockNumber {
			if start.OutputIndex != 1 {
				return b.QueryOutput(ctx, bridgeId, start.OutputIndex-1)
			}
			return nil, nil
		} else if end.OutputProposal.L2BlockNumber < l2BlockNumber {
			return end, nil
		} else if end.OutputIndex-start.OutputIndex <= 1 {
			return start, nil
		}

		midIndex := (start.OutputIndex + end.OutputIndex) / 2
		output, err := b.QueryOutput(ctx, bridgeId, midIndex)
		if err != nil {
			return nil, err
		}

		if output.OutputProposal.L2BlockNumber <= l2BlockNumber {
			start = output
		} else {
			end = output
		}
	}
}

func (b BaseHost) QueryCreateBridgeHeight(ctx context.Context, bridgeId uint64) (uint64, error) {
	ctx, cancel := rpcclient.GetQueryContext(ctx, 0)
	defer cancel()

	query := fmt.Sprintf("%s.%s = %d",
		ophosttypes.EventTypeCreateBridge,
		ophosttypes.AttributeKeyBridgeId,
		bridgeId,
	)
	perPage := 1
	res, err := b.node.GetRPCClient().TxSearch(ctx, query, false, nil, &perPage, "desc")
	if err != nil {
		return 0, err
	}
	if len(res.Txs) == 0 {
		// bridge not found
		return 0, errors.New("bridge not found")
	}
	return uint64(res.Txs[0].Height), nil
}

func (b BaseHost) QueryBatchInfos(ctx context.Context, bridgeId uint64) (*ophosttypes.QueryBatchInfosResponse, error) {
	req := &ophosttypes.QueryBatchInfosRequest{
		BridgeId: bridgeId,
	}
	ctx, cancel := rpcclient.GetQueryContext(ctx, 0)
	defer cancel()
	return b.ophostQueryClient.BatchInfos(ctx, req)
}

func (b BaseHost) QueryDepositTxHeight(ctx context.Context, bridgeId uint64, l1Sequence uint64) (uint64, error) {
	if l1Sequence == 0 {
		return 0, nil
	}

	ctx, cancel := rpcclient.GetQueryContext(ctx, 0)
	defer cancel()

	ticker := time.NewTicker(types.PollingInterval(ctx))
	defer ticker.Stop()

	query := fmt.Sprintf("%s.%s = %d",
		ophosttypes.EventTypeInitiateTokenDeposit,
		ophosttypes.AttributeKeyL1Sequence,
		l1Sequence,
	)
	per_page := 100
	for page := 1; ; page++ {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-ticker.C:
		}

		res, err := b.node.GetRPCClient().TxSearch(ctx, query, false, &page, &per_page, "asc")
		if err != nil {
			return 0, err
		}

		for _, tx := range res.Txs {
			for _, event := range tx.TxResult.Events {
				if event.Type == ophosttypes.EventTypeInitiateTokenDeposit {
					for _, attr := range event.Attributes {
						if attr.Key == ophosttypes.AttributeKeyBridgeId && attr.Value == strconv.FormatUint(bridgeId, 10) {
							return uint64(tx.Height), nil
						}
					}
				}
			}
		}

		if page*per_page >= res.TotalCount {
			break
		}
	}
	return 0, fmt.Errorf("failed to fetch deposit tx with l1 Sequence: %d", l1Sequence)
}