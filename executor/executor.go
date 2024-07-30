package executor

import (
	"context"
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/initia-labs/opinit-bots-go/executor/child"
	"github.com/initia-labs/opinit-bots-go/executor/host"
	"github.com/initia-labs/opinit-bots-go/server"

	bottypes "github.com/initia-labs/opinit-bots-go/bot/types"
	executortypes "github.com/initia-labs/opinit-bots-go/executor/types"
	"github.com/initia-labs/opinit-bots-go/types"
	"go.uber.org/zap"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
)

var _ bottypes.Bot = &Executor{}

// Executor charges the execution of the bridge between the host and the child chain
// - relay l1 deposit messages to l2
// - generate l2 output root and submit to l1
type Executor struct {
	host  *host.Host
	child *child.Child

	cfg    *executortypes.Config
	db     types.DB
	server *server.Server
	logger *zap.Logger
}

func NewExecutor(cfg *executortypes.Config, db types.DB, sv *server.Server, logger *zap.Logger, cdc codec.Codec, txConfig client.TxConfig) *Executor {
	err := cfg.Validate()
	if err != nil {
		panic(err)
	}

	executor := &Executor{
		host: host.NewHost(
			cfg.Version, cfg.RelayOracle, cfg.L1NodeConfig(),
			db.WithPrefix([]byte(executortypes.HostNodeName)),
			logger.Named(executortypes.HostNodeName), cdc, txConfig,
		),
		child: child.NewChild(
			cfg.Version, cfg.L2NodeConfig(),
			db.WithPrefix([]byte(executortypes.ChildNodeName)),
			logger.Named(executortypes.ChildNodeName), cdc, txConfig,
		),

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

	executor.child.Initialize(executor.host, bridgeInfo)
	err = executor.host.Initialize(executor.child, int64(bridgeInfo.BridgeId))
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

	errCh := make(chan error, 3)
	ex.host.Start(hostCtx, errCh)
	ex.child.Start(childCtx, errCh)

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
