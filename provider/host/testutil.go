package host

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"

	"github.com/initia-labs/opinit-bots/node"
	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
)

func NewTestBaseHost(version uint8, node *node.Node, bridgeInfo ophosttypes.QueryBridgeResponse, cfg nodetypes.NodeConfig, ophostQueryClient ophosttypes.QueryClient) *BaseHost {
	return &BaseHost{
		version:           version,
		node:              node,
		bridgeInfo:        bridgeInfo,
		cfg:               cfg,
		ophostQueryClient: ophostQueryClient,
		processedMsgs:     make([]btypes.ProcessedMsgs, 0),
		msgQueue:          make(map[string][]sdk.Msg),
	}
}
