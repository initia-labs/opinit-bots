package challenger

import (
	"context"

	"github.com/pkg/errors"

	"github.com/gofiber/fiber/v2"
	"github.com/initia-labs/opinit-bots/challenger/child"
	"github.com/initia-labs/opinit-bots/challenger/host"
	"github.com/initia-labs/opinit-bots/server"

	bottypes "github.com/initia-labs/opinit-bots/bot/types"
	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"

	"github.com/initia-labs/opinit-bots/types"
	"go.uber.org/zap"
)

var _ bottypes.Bot = &Challenger{}

// Executor charges the execution of the bridge between the host and the child chain
// - relay l1 deposit messages to l2
// - generate l2 output root and submit to l1
type Challenger struct {
	host  *host.Host
	child *child.Child

	cfg    *challengertypes.Config
	db     types.DB
	server *server.Server
	logger *zap.Logger

	homePath string

	elemCh chan challengertypes.ChallengeElem
	elems  map[challengertypes.ChallengeId]map[challengertypes.NodeType]challengertypes.ChallengeElem
}

func NewChallenger(cfg *challengertypes.Config, db types.DB, sv *server.Server, logger *zap.Logger, homePath string) *Challenger {
	err := cfg.Validate()
	if err != nil {
		panic(err)
	}

	elemCh := make(chan challengertypes.ChallengeElem)
	return &Challenger{
		host: host.NewHostV1(
			cfg.L1NodeConfig(homePath),
			db.WithPrefix([]byte(types.HostName)),
			logger.Named(types.HostName), cfg.L1Node.Bech32Prefix,
			elemCh,
		),
		child: child.NewChildV1(
			cfg.L2NodeConfig(homePath),
			db.WithPrefix([]byte(types.ChildName)),
			logger.Named(types.ChildName), cfg.L2Node.Bech32Prefix,
			elemCh,
		),

		cfg:    cfg,
		db:     db,
		server: sv,
		logger: logger,

		homePath: homePath,

		elemCh: elemCh,
		elems:  make(map[challengertypes.ChallengeId]map[challengertypes.NodeType]challengertypes.ChallengeElem),
	}
}

func (c *Challenger) Initialize(ctx context.Context) error {
	bridgeInfo, err := c.child.QueryBridgeInfo(ctx)
	if err != nil {
		return err
	}
	if bridgeInfo.BridgeId == 0 {
		return errors.New("bridge info is not set")
	}

	c.logger.Info(
		"bridge info",
		zap.Uint64("id", bridgeInfo.BridgeId),
		zap.Duration("submission_interval", bridgeInfo.BridgeConfig.SubmissionInterval),
	)

	hostStartHeight, childStartHeight, startOutputIndex, err := c.getStartHeights(ctx, bridgeInfo.BridgeId)
	if err != nil {
		return err
	}

	err = c.host.Initialize(ctx, hostStartHeight, bridgeInfo)
	if err != nil {
		return err
	}
	err = c.child.Initialize(childStartHeight, startOutputIndex, c.host, bridgeInfo)
	if err != nil {
		return err
	}
	c.RegisterQuerier()
	return nil
}

func (c *Challenger) Start(ctx context.Context) error {
	defer c.Close()

	errGrp := types.ErrGrp(ctx)
	errGrp.Go(func() (err error) {
		<-ctx.Done()
		return c.server.Shutdown()
	})

	errGrp.Go(func() (err error) {
		defer func() {
			c.logger.Info("api server stopped")
		}()
		return c.server.Start(c.cfg.ListenAddress)
	})

	errGrp.Go(func() (err error) {
		defer func() {
			c.logger.Info("challenge handler stopped")
		}()
		return c.challengeHandler(ctx)
	})

	// TODO: load elems and send them to elemch first

	c.host.Start(ctx)
	c.child.Start(ctx)
	return errGrp.Wait()
}

func (c *Challenger) Close() {
	c.db.Close()
}

func (c *Challenger) RegisterQuerier() {
	c.server.RegisterQuerier("/status", func(ctx *fiber.Ctx) error {
		return ctx.JSON(c.GetStatus())
	})
}

func (c *Challenger) getStartHeights(ctx context.Context, bridgeId uint64) (l1StartHeight uint64, l2StartHeight uint64, startOutputIndex uint64, err error) {
	// get the bridge start height from the host
	l1StartHeight, err = c.host.QueryCreateBridgeHeight(ctx, bridgeId)
	if err != nil {
		return 0, 0, 0, err
	}

	// get the last submitted output height before the start height from the host
	if c.cfg.L2StartHeight != 0 {
		output, err := c.host.QueryLastFinalizedOutput(ctx, bridgeId)
		if err != nil {
			return 0, 0, 0, err
		} else if output != nil {
			l1StartHeight = output.OutputProposal.L1BlockNumber
			l2StartHeight = output.OutputProposal.L2BlockNumber
			startOutputIndex = output.OutputIndex + 1
		}
	}
	if l2StartHeight > 0 {
		// get the last deposit tx height from the host
		l1Sequence, err := c.child.QueryNextL1Sequence(ctx, l2StartHeight-1)
		if err != nil {
			return 0, 0, 0, err
		}
		depositTxHeight, err := c.host.QueryDepositTxHeight(ctx, bridgeId, l1Sequence-1)
		if err != nil {
			return 0, 0, 0, err
		}
		if l1StartHeight > depositTxHeight {
			l1StartHeight = depositTxHeight
		}
	}
	if l2StartHeight == 0 {
		startOutputIndex = 1
	}
	return l1StartHeight, l2StartHeight, startOutputIndex, err
}
