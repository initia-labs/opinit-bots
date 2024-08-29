package host

import (
	"context"

	"go.uber.org/zap"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"

	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"

	hostprovider "github.com/initia-labs/opinit-bots/provider/host"

	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
)

type Host struct {
	*hostprovider.BaseHost

	elemCh    chan<- challengertypes.ChallengeElem
	elemQueue []challengertypes.ChallengeElem
}

func NewHostV1(
	cfg nodetypes.NodeConfig,
	db types.DB, logger *zap.Logger, bech32Prefix string,
	elemCh chan<- challengertypes.ChallengeElem,
) *Host {
	return &Host{
		BaseHost:  hostprovider.NewBaseHostV1(cfg, db, logger, bech32Prefix),
		elemCh:    elemCh,
		elemQueue: make([]challengertypes.ChallengeElem, 0),
	}
}

func (h *Host) Initialize(ctx context.Context, startHeight uint64, bridgeId int64) error {
	err := h.BaseHost.Initialize(ctx, startHeight, bridgeId)
	if err != nil {
		return err
	}
	// TODO: ignore l1Sequence less than child's last l1 sequence
	h.registerHandlers()
	return nil
}

func (h *Host) registerHandlers() {
	h.Node().RegisterBeginBlockHandler(h.beginBlockHandler)
	h.Node().RegisterEventHandler(ophosttypes.EventTypeInitiateTokenDeposit, h.initiateDepositHandler)
	h.Node().RegisterEventHandler(ophosttypes.EventTypeProposeOutput, h.proposeOutputHandler)
	h.Node().RegisterEndBlockHandler(h.endBlockHandler)
}

func (h Host) NodeType() challengertypes.NodeType {
	return challengertypes.NodeTypeHost
}
