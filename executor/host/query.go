package host

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	query "github.com/cosmos/cosmos-sdk/types/query"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
)

func (h Host) GetAddress() sdk.AccAddress {
	return h.node.GetAddress()
}

func (h Host) QueryLastOutput() (ophosttypes.QueryOutputProposalResponse, error) {
	req := &ophosttypes.QueryOutputProposalsRequest{
		BridgeId: uint64(h.bridgeId),
		Pagination: &query.PageRequest{
			Limit:   1,
			Reverse: true,
		},
	}
	res, err := h.ophostQueryClient.OutputProposals(context.Background(), req)
	if err != nil {
		return ophosttypes.QueryOutputProposalResponse{}, err
	}
	return res.OutputProposals[0], nil
}
