package host

import (
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
)

type Status struct {
	Node                            nodetypes.Status `json:"node"`
	LastProposedOutputIndex         uint64           `json:"last_proposed_output_index"`
	LastProposedOutputL2BlockNumber uint64           `json:"last_proposed_output_l2_block_number"`
}

func (h Host) GetStatus() Status {
	return Status{
		Node:                            h.GetNodeStatus(),
		LastProposedOutputIndex:         h.lastProposedOutputIndex,
		LastProposedOutputL2BlockNumber: h.lastProposedOutputL2BlockNumber,
	}
}

func (h Host) GetNodeStatus() nodetypes.Status {
	return h.Node().GetStatus()
}
