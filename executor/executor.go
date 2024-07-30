package executor

import (
	"context"
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/initia-labs/opinit-bots-go/executor/batch"
	"github.com/initia-labs/opinit-bots-go/executor/child"
	"github.com/initia-labs/opinit-bots-go/executor/host"
	"github.com/initia-labs/opinit-bots-go/server"

	bottypes "github.com/initia-labs/opinit-bots-go/bot/types"
	executortypes "github.com/initia-labs/opinit-bots-go/executor/types"
	"github.com/initia-labs/opinit-bots-go/types"
	"go.uber.org/zap"
)

var _ bottypes.Bot = &Executor{}

// Executor charges the execution of the bridge between the host and the child chain
// - relay l1 deposit messages to l2
// - generate l2 output root and submit to l1
type Executor struct {
	host  *host.Host
	child *child.Child
	batch *batch.BatchSubmitter

	cfg    *executortypes.Config
	db     types.DB
	server *server.Server
	logger *zap.Logger
}

func NewExecutor(cfg *executortypes.Config, db types.DB, sv *server.Server, logger *zap.Logger, homePath string) *Executor {
	err := cfg.Validate()
	if err != nil {
		panic(err)
	}

	executor := &Executor{
		host: host.NewHost(
			cfg.Version, cfg.RelayOracle, cfg.L1NodeConfig(),
			db.WithPrefix([]byte(executortypes.HostNodeName)),
			logger.Named(executortypes.HostNodeName),
		),
		child: child.NewChild(
			cfg.Version, cfg.L2NodeConfig(),
			db.WithPrefix([]byte(executortypes.ChildNodeName)),
			logger.Named(executortypes.ChildNodeName),
		),
		batch: batch.NewBatchSubmitter(cfg.Version, cfg.L2NodeConfig(), cfg.BatchConfig(), db.WithPrefix([]byte(executortypes.BatchNodeName)), logger.Named(executortypes.BatchNodeName), homePath),

		cfg:    cfg,
		db:     db,
		server: sv,
		logger: logger,
	}

	bridgeInfo, err := executor.child.QueryBridgeInfo()
	if err != nil {
		panic(err)
	}
	if bridgeInfo.BridgeId == 0 {
		panic("bridge info is not set")
	}

	executor.logger.Info(
		"bridge info",
		zap.Uint64("id", bridgeInfo.BridgeId),
		zap.Duration("submission_interval", bridgeInfo.BridgeConfig.SubmissionInterval),
	)

	da := executor.host
	// if cfg.Batch.DANode.ChainID != cfg.HostNode.ChainID {
	// 	switch cfg.Batch.DANode.ChainID {
	// 	case "celestia":
	// 		da = celestia.NewCelestia()
	// 	}
	// }

	err = executor.host.Initialize(executor.child, executor.batch, int64(bridgeInfo.BridgeId))
	if err != nil {
		panic(err)
	}
	err = executor.child.Initialize(executor.host, bridgeInfo)
	if err != nil {
		panic(err)
	}
	err = executor.batch.Initialize(executor.host, da, bridgeInfo)
	if err != nil {
		panic(err)
	}
	executor.RegisterQuerier()
	return executor
}

func (ex *Executor) Start(cmdCtx context.Context) error {
	defer ex.db.Close()

	err := ex.server.ShutdownWithContext(cmdCtx)
	if err != nil {
		return err
	}

	hostCtx, hostDone := context.WithCancel(cmdCtx)
	childCtx, childDone := context.WithCancel(cmdCtx)
	batchCtx, batchDone := context.WithCancel(cmdCtx)

	errCh := make(chan error, 3)
	ex.host.Start(hostCtx, errCh)
	ex.child.Start(childCtx, errCh)
	ex.batch.Start(batchCtx, errCh)

	go func() {
		err := ex.server.Start(ex.cfg.ListenAddress)
		if err != nil {
			errCh <- err
		}
	}()

	shutdown := func(err error) error {
		ex.logger.Info("executor shutdown", zap.String("state", "requested"))

		ex.logger.Debug("executor shutdown", zap.String("state", "wait"), zap.String("target", "api"))
		ex.server.Shutdown()

		ex.logger.Debug("executor shutdown", zap.String("state", "wait"), zap.String("target", "host"))
		hostDone()

		ex.logger.Debug("executor shutdown", zap.String("state", "wait"), zap.String("target", "child"))
		childDone()

		ex.logger.Debug("executor shutdown", zap.String("state", "wait"), zap.String("target", "batch"))
		batchDone()

		ex.logger.Info("executor shutdown completed")
		return err
	}

	select {
	case err := <-errCh:
		ex.logger.Error("executor error", zap.String("error", err.Error()))
		return shutdown(err)
	case <-cmdCtx.Done():
		return shutdown(nil)
	}
}

func (ex *Executor) RegisterQuerier() {
	ex.server.RegisterQuerier("/withdrawal/:sequence", func(c *fiber.Ctx) error {
		sequenceStr := c.Params("sequence")
		if sequenceStr == "" {
			return errors.New("sequence is required")
		}
		sequence, err := strconv.ParseUint(sequenceStr, 10, 64)
		if err != nil {
			return err
		}
		res, err := ex.child.QueryWithdrawal(sequence)
		if err != nil {
			return err
		}
		return c.JSON(res)
	})

	ex.server.RegisterQuerier("/status", func(c *fiber.Ctx) error {
		childHeight := ex.child.GetHeight()
		hostHeight := ex.host.GetHeight()
		res := map[string]uint64{
			"child": childHeight,
			"host":  hostHeight,
		}

		return c.JSON(res)
	})
}
