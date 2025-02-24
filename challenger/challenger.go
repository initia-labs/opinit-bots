package challenger

import (
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/gofiber/fiber/v2"
	"github.com/initia-labs/opinit-bots/challenger/child"
	challengerdb "github.com/initia-labs/opinit-bots/challenger/db"
	"github.com/initia-labs/opinit-bots/challenger/host"
	"github.com/initia-labs/opinit-bots/sentry_integration"
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

	challengeCh        chan challengertypes.Challenge
	challengeChStopped chan struct{}

	pendingChallenges []challengertypes.Challenge

	// status info
	latestChallengesMu *sync.Mutex
	latestChallenges   []challengertypes.Challenge

	stage types.CommitDB
}

func NewChallenger(cfg *challengertypes.Config, db types.DB, sv *server.Server) *Challenger {
	err := sentry_integration.Init("opinit-bots", string(bottypes.BotTypeChallenger))
	if err != nil {
		panic(err)
	}

	err = cfg.Validate()
	if err != nil {
		panic(err)
	}

	challengeCh := make(chan challengertypes.Challenge)
	return &Challenger{
		host: host.NewHostV1(
			cfg.L1NodeConfig(),
			db.WithPrefix([]byte(types.HostName)),
		),
		child: child.NewChildV1(
			cfg.L2NodeConfig(),
			db.WithPrefix([]byte(types.ChildName)),
		),

		cfg:    cfg,
		db:     db,
		server: sv,

		challengeCh:        challengeCh,
		challengeChStopped: make(chan struct{}),

		pendingChallenges: make([]challengertypes.Challenge, 0),

		latestChallengesMu: &sync.Mutex{},
		latestChallenges:   make([]challengertypes.Challenge, 0),

		stage: db.NewStage(),
	}
}

func (c *Challenger) Initialize(ctx types.Context) error {
	childBridgeInfo, err := c.child.QueryBridgeInfo(ctx)
	if err != nil {
		return err
	}
	if childBridgeInfo.BridgeId == 0 {
		return errors.New("bridge info is not set")
	}

	bridgeInfo, err := c.host.QueryBridgeConfig(ctx, childBridgeInfo.BridgeId)
	if err != nil {
		return err
	}

	ctx.Logger().Info(
		"bridge info",
		zap.Uint64("id", bridgeInfo.BridgeId),
		zap.Duration("submission_interval", bridgeInfo.BridgeConfig.SubmissionInterval),
	)

	childBridgeInfo.BridgeConfig = bridgeInfo.BridgeConfig

	l1StartHeight, l2StartHeight, startOutputIndex, err := c.getNodeStartHeights(ctx, bridgeInfo.BridgeId)
	if err != nil {
		return err
	}

	var initialBlockTime time.Time
	hostInitialBlockTime, err := c.host.Initialize(ctx, l1StartHeight-1, c.child, *bridgeInfo, c)
	if err != nil {
		return err
	}
	if initialBlockTime.Before(hostInitialBlockTime) {
		initialBlockTime = hostInitialBlockTime
	}

	childInitialBlockTime, err := c.child.Initialize(ctx, l2StartHeight-1, startOutputIndex, c.host, childBridgeInfo, c)
	if err != nil {
		return err
	}
	if initialBlockTime.Before(childInitialBlockTime) {
		initialBlockTime = childInitialBlockTime
	}

	// only called when `ResetHeight` was executed.
	if !initialBlockTime.IsZero() {
		// The db state is reset to a specific height, so we also
		// need to delete future challenges which are not applicable anymore.
		err := challengerdb.DeleteFutureChallenges(c.db, initialBlockTime)
		if err != nil {
			return err
		}
	}

	c.RegisterQuerier(ctx)

	c.pendingChallenges, err = challengerdb.LoadPendingChallenges(c.db)
	if err != nil {
		return errors.Wrap(err, "failed to load pending challenges")
	}

	c.latestChallenges, err = challengerdb.LoadChallenges(c.db, 5)
	if err != nil {
		return errors.Wrap(err, "failed to load challenges")
	}

	return nil
}

func (c *Challenger) Start(ctx types.Context) error {
	ctx.ErrGrp().Go(func() (err error) {
		<-ctx.Done()
		return c.server.Shutdown()
	})

	ctx.ErrGrp().Go(func() (err error) {
		defer func() {
			ctx.Logger().Info("api server stopped")
		}()
		return c.server.Start()
	})

	ctx.ErrGrp().Go(func() error {
		for _, ch := range c.pendingChallenges {
			c.challengeCh <- ch
		}
		return nil
	})

	ctx.ErrGrp().Go(func() (err error) {
		defer func() {
			ctx.Logger().Info("challenge handler stopped")
		}()
		return c.challengeHandler(ctx)
	})

	c.host.Start(ctx)
	c.child.Start(ctx)
	return ctx.ErrGrp().Wait()
}

func (c *Challenger) RegisterQuerier(ctx types.Context) {
	c.server.RegisterQuerier("/status", func(fiberCtx *fiber.Ctx) error {
		status, err := c.GetStatus()
		if err != nil {
			ctx.Logger().Error("failed to query status", zap.Error(err))
			return errors.New("failed to query status")
		}

		return fiberCtx.JSON(status)
	})
	c.server.RegisterQuerier("/challenges", func(fiberCtx *fiber.Ctx) error {
		offset := fiberCtx.Query("offset", "")
		limit := fiberCtx.QueryInt("limit", 10)
		if limit > 100 {
			limit = 100
		}

		ulimit, err := types.SafeInt64ToUint64(int64(limit))
		if err != nil {
			return errors.Wrap(err, "failed to convert limit")
		}

		descOrder := true
		orderStr := fiberCtx.Query("order", "desc")
		if orderStr == "asc" {
			descOrder = false
		}

		res, err := c.QueryChallenges(offset, ulimit, descOrder)
		if err != nil {
			ctx.Logger().Error("failed to query challenges", zap.Error(err))
			return errors.New("failed to query challenges")
		}
		return fiberCtx.JSON(res)
	})

	c.server.RegisterQuerier("/pending_events/host", func(fiberCtx *fiber.Ctx) error {
		pendingEvents, err := c.host.GetAllPendingEvents()
		if err != nil {
			ctx.Logger().Error("failed to query pending events", zap.Error(err))
			return errors.New("failed to query pending events")
		}
		return fiberCtx.JSON(pendingEvents)
	})

	c.server.RegisterQuerier("/pending_events/child", func(fiberCtx *fiber.Ctx) error {
		pendingEvents, err := c.child.GetAllPendingEvents()
		if err != nil {
			ctx.Logger().Error("failed to query pending events", zap.Error(err))
			return errors.New("failed to query pending events")
		}
		return fiberCtx.JSON(pendingEvents)
	})

	c.server.RegisterQuerier("/syncing", func(fiberCtx *fiber.Ctx) error {
		status, err := c.GetStatus()
		if err != nil {
			ctx.Logger().Error("failed to query status", zap.Error(err))
			return errors.New("failed to query status")
		}
		hostSync := status.Host.Node.Syncing != nil && *status.Host.Node.Syncing
		childSync := status.Child.Node.Syncing != nil && *status.Child.Node.Syncing
		return fiberCtx.JSON(hostSync || childSync)
	})
}

func (c *Challenger) getNodeStartHeights(ctx types.Context, bridgeId uint64) (l1StartHeight int64, l2StartHeight int64, startOutputIndex uint64, err error) {
	if c.host.Node().GetSyncedHeight() != 0 && c.child.Node().GetSyncedHeight() != 0 {
		return 0, 0, 0, nil
	}

	var outputL1Height, outputL2Height int64
	var outputIndex uint64

	// get the last submitted output height before the start height from the host
	if c.cfg.L2StartHeight > 1 {
		output, err := c.host.QueryLastFinalizedOutput(ctx, bridgeId)
		if err != nil {
			return 0, 0, 0, err
		} else if output != nil {
			outputL1Height = types.MustUint64ToInt64(output.OutputProposal.L1BlockNumber)
			outputL2Height = types.MustUint64ToInt64(output.OutputProposal.L2BlockNumber)
			outputIndex = output.OutputIndex
		}
	}
	l2StartHeight = outputL2Height + 1
	startOutputIndex = outputIndex + 1

	if c.cfg.DisableAutoSetL1Height {
		l1StartHeight = c.cfg.L1StartHeight
	} else {
		if l2StartHeight > 1 {
			l1Sequence, err := c.child.QueryNextL1Sequence(ctx, l2StartHeight-1)
			if err != nil {
				return 0, 0, 0, err
			}
			// query l1Sequence tx height
			depositTxHeight, err := c.host.QueryDepositTxHeight(ctx, bridgeId, l1Sequence)
			if err != nil {
				return 0, 0, 0, err
			} else if depositTxHeight == 0 && l1Sequence > 1 {
				// query l1Sequence - 1 tx height
				depositTxHeight, err = c.host.QueryDepositTxHeight(ctx, bridgeId, l1Sequence-1)
				if err != nil {
					return 0, 0, 0, err
				}
			}

			if depositTxHeight > l1StartHeight {
				l1StartHeight = depositTxHeight
			}
			if outputL1Height != 0 && outputL1Height < l1StartHeight {
				l1StartHeight = outputL1Height + 1
			}
		}

		// if l1 start height is not set, get the bridge start height from the host
		if l1StartHeight == 0 {
			// get the bridge start height from the host
			l1StartHeight, err = c.host.QueryCreateBridgeHeight(ctx, bridgeId)
			if err != nil {
				return 0, 0, 0, err
			}
		}
	}
	return
}

func (c Challenger) DB() types.DB {
	return c.db
}
