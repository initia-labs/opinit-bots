package child

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"

	"github.com/initia-labs/opinit-bots/merkle"
	"github.com/initia-labs/opinit-bots/node"
	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
)

func NewTestBaseChild(
	version uint8,

	node *node.Node,
	mk *merkle.Merkle,

	bridgeInfo ophosttypes.QueryBridgeResponse,

	initializeTreeFn func(int64) (bool, error),

	cfg nodetypes.NodeConfig,
) *BaseChild {
	return &BaseChild{
		version: version,

		node: node,
		mk:   mk,

		bridgeInfo: bridgeInfo,

		initializeTreeFn: initializeTreeFn,

		cfg: cfg,

		processedMsgs: make([]btypes.ProcessedMsgs, 0),
		msgQueue:      make(map[string][]sdk.Msg),
	}
}
