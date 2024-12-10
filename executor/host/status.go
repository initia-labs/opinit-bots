package host

import (
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/pkg/errors"
)

type Status struct {
	Node                            nodetypes.Status `json:"node"`
	LastProposedOutputIndex         uint64           `json:"last_proposed_output_index"`
	LastProposedOutputL2BlockNumber int64            `json:"last_proposed_output_l2_block_number"`
}

func (h Host) GetStatus() (Status, error) {
	nodeStatus, err := h.GetNodeStatus()
	if err != nil {
		return Status{}, errors.Wrap(err, "failed to get node status")
	}

	return Status{
		Node:                            nodeStatus,
		LastProposedOutputIndex:         h.lastProposedOutputIndex,
		LastProposedOutputL2BlockNumber: h.lastProposedOutputL2BlockNumber,
	}, nil
}

func (h Host) GetNodeStatus() (nodetypes.Status, error) {
	if h.Node() == nil {
		return nodetypes.Status{}, errors.New("node is not initialized")
	}
	return h.Node().GetStatus(), nil
}
