package host

import (
	"errors"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	query "github.com/cosmos/cosmos-sdk/types/query"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"

	"github.com/initia-labs/opinit-bots-go/node/rpcclient"
)

func (h Host) GetAddress() sdk.AccAddress {
	return h.node.MustGetBroadcaster().GetAddress()
}

func (h Host) GetAddressStr() (string, error) {
	return h.node.MustGetBroadcaster().GetAddressString()
}

func (h Host) QueryLastOutput(bridgeId uint64) (*ophosttypes.QueryOutputProposalResponse, error) {
	req := &ophosttypes.QueryOutputProposalsRequest{
		BridgeId: bridgeId,
		Pagination: &query.PageRequest{
			Limit:   1,
			Reverse: true,
		},
	}
	ctx, cancel := rpcclient.GetQueryContext(0)
	defer cancel()

	res, err := h.ophostQueryClient.OutputProposals(ctx, req)
	if err != nil {
		return nil, err
	}
	if res.OutputProposals == nil || len(res.OutputProposals) == 0 {
		return nil, nil
	}
	return &res.OutputProposals[0], nil
}

func (h Host) QueryOutput(bridgeId uint64, outputIndex uint64) (*ophosttypes.QueryOutputProposalResponse, error) {
	req := &ophosttypes.QueryOutputProposalRequest{
		BridgeId:    bridgeId,
		OutputIndex: outputIndex,
	}
	ctx, cancel := rpcclient.GetQueryContext(0)
	defer cancel()

	return h.ophostQueryClient.OutputProposal(ctx, req)
}

// QueryOutputByL2BlockNumber queries the last output proposal before the given L2 block number
func (h Host) QueryOutputByL2BlockNumber(bridgeId uint64, l2BlockNumber uint64) (*ophosttypes.QueryOutputProposalResponse, error) {
	start, err := h.QueryOutput(bridgeId, 1)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, nil
		}
		return nil, err
	}
	end, err := h.QueryLastOutput(bridgeId)
	if err != nil {
		return nil, err
	} else if end == nil {
		return nil, nil
	}

	for {
		if start.OutputProposal.L2BlockNumber >= l2BlockNumber {
			return nil, nil
		} else if end.OutputProposal.L2BlockNumber < l2BlockNumber {
			return end, nil
		} else if end.OutputIndex-start.OutputIndex <= 1 {
			return start, nil
		}

		midIndex := (start.OutputIndex + end.OutputIndex) / 2
		output, err := h.QueryOutput(bridgeId, midIndex)
		if err != nil {
			return nil, err
		}

		if output.OutputProposal.L2BlockNumber == l2BlockNumber {
			return output, nil
		} else if output.OutputProposal.L2BlockNumber < l2BlockNumber {
			start = output
		} else {
			end = output
		}
	}
}

func (h Host) QueryCreateBridgeHeight(bridgeId uint64) (uint64, error) {
	ctx, cancel := rpcclient.GetQueryContext(0)
	defer cancel()

	query := fmt.Sprintf("%s.%s = %d",
		ophosttypes.EventTypeCreateBridge,
		ophosttypes.AttributeKeyBridgeId,
		bridgeId,
	)
	perPage := 1
	res, err := h.node.GetRPCClient().TxSearch(ctx, query, false, nil, &perPage, "desc")
	if err != nil {
		return 0, err
	}
	if len(res.Txs) == 0 {
		// bridge not found
		return 0, errors.New("bridge not found")
	}
	return uint64(res.Txs[0].Height), nil
}

func (h Host) QueryBatchInfos(bridgeId uint64) (*ophosttypes.QueryBatchInfosResponse, error) {
	req := &ophosttypes.QueryBatchInfosRequest{
		BridgeId: bridgeId,
	}
	ctx, cancel := rpcclient.GetQueryContext(0)
	defer cancel()
	return h.ophostQueryClient.BatchInfos(ctx, req)
}
