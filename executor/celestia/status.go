package celestia

import (
	"errors"

	nodetypes "github.com/initia-labs/opinit-bots/node/types"
)

func (c Celestia) GetNodeStatus() (nodetypes.Status, error) {
	if c.node == nil {
		return nodetypes.Status{}, errors.New("node is not initialized")
	}
	return c.node.GetStatus(), nil
}
