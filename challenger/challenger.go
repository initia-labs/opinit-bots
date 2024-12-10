package challenger

import (
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/gofiber/fiber/v2"
	"github.com/initia-labs/opinit-bots/challenger/child"
	challengerdb "github.com/initia-labs/opinit-bots/challenger/db"
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

	challengeCh        chan challengertypes.Challenge
	challengeChStopped chan struct{}

	pendingChallenges []challengertypes.Challenge

	// status info
	latestChallengesMu *sync.Mutex
	latestChallenges   []challengertypes.Challenge

	stage types.CommitDB
}

func NewChallenger(cfg *challengertypes.Config, db types.DB, sv *server.Server) *Challenger {
	err := cfg.Validate()
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

	hostProcessedHeight, childProcessedHeight, processedOutputIndex, err := c.getProcessedHeights(ctx, bridgeInfo.BridgeId)
	if err != nil {
		return err
	}

	var initialBlockTime time.Time
	hostInitialBlockTime, err := c.host.Initialize(ctx, hostProcessedHeight, c.child, *bridgeInfo, c)
	if err != nil {
		return err
	}
	if initialBlockTime.Before(hostInitialBlockTime) {
		initialBlockTime = hostInitialBlockTime
	}

	childInitialBlockTime, err := c.child.Initialize(ctx, childProcessedHeight, processedOutputIndex+1, c.host, *bridgeInfo, c)
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

	c.RegisterQuerier()

	c.pendingChallenges, err = challengerdb.LoadPendingChallenges(c.db)
	if err != nil {
		return errors.Wrap(err, "failed to load pending challenges")
	}

	c.latestChallenges, err = challengerdb.LoadChallenges(c.db)
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

func (c *Challenger) RegisterQuerier() {
	c.server.RegisterQuerier("/status", func(ctx *fiber.Ctx) error {
		status, err := c.GetStatus()
		if err != nil {
			return err
		}

		return ctx.JSON(status)
	})
	c.server.RegisterQuerier("/challenges", func(ctx *fiber.Ctx) error {
		next := ctx.Query("next", "")
		limit := ctx.QueryInt("limit", 10)
		if limit > 100 {
			limit = 100
		}

		ulimit, err := types.SafeInt64ToUint64(int64(limit))
		if err != nil {
			return errors.Wrap(err, "failed to convert limit")
		}

		descOrder := true
		orderStr := ctx.Query("order", "desc")
		if orderStr == "asc" {
			descOrder = false
		}

		res, err := c.QueryChallenges(next, ulimit, descOrder)
		if err != nil {
			return err
		}
		return ctx.JSON(res)
	})

	c.server.RegisterQuerier("/pending_events/host", func(ctx *fiber.Ctx) error {
		pendingEvents, err := c.host.GetAllPendingEvents()
		if err != nil {
			return err
		}
		return ctx.JSON(pendingEvents)
	})

	c.server.RegisterQuerier("/pending_events/child", func(ctx *fiber.Ctx) error {
		pendingEvents, err := c.child.GetAllPendingEvents()
		if err != nil {
			return err
		}
		return ctx.JSON(pendingEvents)
	})
}

func (c *Challenger) getProcessedHeights(ctx types.Context, bridgeId uint64) (l1ProcessedHeight int64, l2ProcessedHeight int64, processedOutputIndex uint64, err error) {
	if c.host.Node().GetSyncedHeight() != 0 && c.child.Node().GetSyncedHeight() != 0 {
		return 0, 0, 0, nil
	}

	var outputL1BlockNumber int64
	// get the last submitted output height before the start height from the host
	if c.cfg.L2StartHeight != 0 {
		output, err := c.host.QueryLastFinalizedOutput(ctx, bridgeId)
		if err != nil {
			return 0, 0, 0, err
		} else if output != nil {
			outputL1BlockNumber = types.MustUint64ToInt64(output.OutputProposal.L1BlockNumber)
			l2ProcessedHeight = types.MustUint64ToInt64(output.OutputProposal.L2BlockNumber)
			processedOutputIndex = output.OutputIndex
		}
	}

	if c.cfg.DisableAutoSetL1Height {
		l1ProcessedHeight = c.cfg.L1StartHeight
	} else {
		// get the bridge start height from the host
		l1ProcessedHeight, err = c.host.QueryCreateBridgeHeight(ctx, bridgeId)
		if err != nil {
			return 0, 0, 0, err
		}

		if l2ProcessedHeight > 0 {
			l1Sequence, err := c.child.QueryNextL1Sequence(ctx, l2ProcessedHeight-1)
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

			if depositTxHeight > l1ProcessedHeight {
				l1ProcessedHeight = depositTxHeight
			}
			if outputL1BlockNumber != 0 && outputL1BlockNumber < l1ProcessedHeight {
				l1ProcessedHeight = outputL1BlockNumber
			}
		}
	}
	if l1ProcessedHeight > 0 {
		l1ProcessedHeight--
	}

	return l1ProcessedHeight, l2ProcessedHeight, processedOutputIndex, err
}

func (c Challenger) DB() types.DB {
	return c.db
}
