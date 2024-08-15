package celestia

import (
	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"
)

func (c Celestia) GetNodeStatus() nodetypes.Status {
	return c.node.GetStatus()
}
