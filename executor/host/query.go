package host

import (
	"errors"

	query "github.com/cosmos/cosmos-sdk/types/query"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	"github.com/initia-labs/opinit-bots-go/node"
)

func (h Host) GetAddressStr() (string, error) {
	addr := h.node.GetAddress()
	if addr == nil {
		return "", errors.New("nil address")
	}
	return h.ac.BytesToString(addr)
}

func (h Host) QueryLastOutput() (ophosttypes.QueryOutputProposalResponse, error) {
	req := &ophosttypes.QueryOutputProposalsRequest{
		BridgeId: uint64(h.bridgeId),
		Pagination: &query.PageRequest{
			Limit:   1,
			Reverse: true,
		},
	}
	ctx := node.GetQueryContext(0)
	res, err := h.ophostQueryClient.OutputProposals(ctx, req)
	if err != nil {
		return ophosttypes.QueryOutputProposalResponse{}, err
	}
	if res.OutputProposals == nil || len(res.OutputProposals) == 0 {
		return ophosttypes.QueryOutputProposalResponse{}, nil
	}
	return res.OutputProposals[0], nil
}
