package challenger

import (
	"context"

	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	"github.com/initia-labs/opinit-bots/types"
	"go.uber.org/zap"
)

type Challenger struct {
}

func NewChallenger(cfg *challengertypes.Config, db types.DB, logger *zap.Logger, homePath string) *Challenger {
	err := cfg.Validate()
	if err != nil {
		panic(err)
	}

	return &Challenger{}
}

func (c *Challenger) Initialize(ctx context.Context) error {

	return nil
}
