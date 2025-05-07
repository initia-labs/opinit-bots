package celestia

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"

	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"

	executortypes "github.com/initia-labs/opinit-bots/executor/types"
)

func (c Celestia) GetNodeStatus() (nodetypes.Status, error) {
	if c.node == nil {
		return nodetypes.Status{}, errors.New("node is not initialized")
	}
	return c.node.GetStatus(), nil
}

type InternalStatus struct {
	LastUpdatedBatchTime time.Time `json:"last_updated_batch_time"`
}

func (c Celestia) GetInternalStatus() InternalStatus {
	return InternalStatus{
		LastUpdatedBatchTime: c.lastUpdatedBatchTime,
	}
}

func (c Celestia) SaveInternalStatus(db types.BasicDB) error {
	internalStatusBytes, err := json.Marshal(c.GetInternalStatus())
	if err != nil {
		return errors.Wrap(err, "failed to marshal internal status")
	}
	return db.Set(executortypes.InternalStatusKey, internalStatusBytes)
}

func (c *Celestia) LoadInternalStatus() error {
	internalStatusBytes, err := c.DB().Get(executortypes.InternalStatusKey)
	if err != nil {
		return errors.Wrap(err, "failed to get internal status")
	}
	var internalStatus InternalStatus
	err = json.Unmarshal(internalStatusBytes, &internalStatus)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal internal status")
	}
	c.lastUpdatedBatchTime = internalStatus.LastUpdatedBatchTime
	return nil
}
