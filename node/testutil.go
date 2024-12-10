package node

import (
	"sync"

	"github.com/initia-labs/opinit-bots/node/broadcaster"
	"github.com/initia-labs/opinit-bots/node/rpcclient"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
)

func NewTestNode(
	cfg nodetypes.NodeConfig,
	db types.DB,

	cdc codec.Codec,
	txConfig client.TxConfig,

	rpcClient *rpcclient.RPCClient,
	broadcaster *broadcaster.Broadcaster,
) *Node {
	return &Node{
		rpcClient:   rpcClient,
		broadcaster: broadcaster,

		cfg: cfg,
		db:  db,

		eventHandlers: make(map[string]nodetypes.EventHandlerFn),

		cdc:      cdc,
		txConfig: txConfig,

		startOnce: &sync.Once{},
	}
}
