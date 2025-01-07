package types

import (
	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
)

type Status struct {
	Syncing         *bool                     `json:"syncing,omitempty"`
	LastBlockHeight *int64                    `json:"last_block_height,omitempty"`
	Broadcaster     *btypes.BroadcasterStatus `json:"broadcaster,omitempty"`
}
