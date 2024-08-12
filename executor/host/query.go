package host

import (
	"fmt"
	"strconv"

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

func (h Host) QueryLastOutput() (*ophosttypes.QueryOutputProposalResponse, error) {
	req := &ophosttypes.QueryOutputProposalsRequest{
		BridgeId: uint64(h.bridgeId),
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

func (h Host) QueryOutput(outputIndex uint64) (*ophosttypes.QueryOutputProposalResponse, error) {
	req := &ophosttypes.QueryOutputProposalRequest{
		BridgeId:    uint64(h.bridgeId),
		OutputIndex: outputIndex,
	}
	ctx, cancel := rpcclient.GetQueryContext(0)
	defer cancel()

	return h.ophostQueryClient.OutputProposal(ctx, req)
}

func (h Host) QueryBatchInfos() (*ophosttypes.QueryBatchInfosResponse, error) {
	req := &ophosttypes.QueryBatchInfosRequest{
		BridgeId: uint64(h.bridgeId),
	}
	ctx, cancel := rpcclient.GetQueryContext(0)
	defer cancel()
	return h.ophostQueryClient.BatchInfos(ctx, req)
}

func (h Host) QueryHeightsOfOutputTxWithL2BlockNumber(bridgeId int64, l2BlockNumber uint64) (uint64, uint64, uint64, error) {
	ctx, cancel := rpcclient.GetQueryContext(0)
	defer cancel()

	query := fmt.Sprintf("%s.%s = %d AND %s.%s <= %d", ophosttypes.EventTypeProposeOutput,
		ophosttypes.AttributeKeyBridgeId,
		bridgeId,
		ophosttypes.EventTypeProposeOutput,
		ophosttypes.AttributeKeyL2BlockNumber,
		l2BlockNumber,
	)
	perPage := 1
	res, err := h.node.GetRPCClient().TxSearch(ctx, query, false, nil, &perPage, "desc")
	if err != nil {
		return 0, 0, 0, err
	}
	if len(res.Txs) == 0 {
		// no output tx found
		return 0, 0, 1, nil
	}

	l2StartHeight := uint64(0)
	outputIndex := uint64(0)
	for _, event := range res.Txs[0].TxResult.Events {
		if event.Type == ophosttypes.EventTypeProposeOutput {
			for _, attr := range event.Attributes {
				if attr.Key == ophosttypes.AttributeKeyL2BlockNumber {
					l2StartHeight, err = strconv.ParseUint(attr.Value, 10, 64)
					if err != nil {
						return 0, 0, 0, err
					}
				} else if attr.Key == ophosttypes.AttributeKeyOutputIndex {
					outputIndex, err = strconv.ParseUint(attr.Value, 10, 64)
					if err != nil {
						return 0, 0, 0, err
					}
				}
			}
		}
	}
	if l2StartHeight == 0 || outputIndex == 0 {
		return 0, 0, 0, fmt.Errorf("something wrong: l2 block number not found in the output tx")
	}
	return uint64(res.Txs[0].Height), l2StartHeight, outputIndex + 1, nil
}
