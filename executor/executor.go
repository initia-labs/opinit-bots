package executor

import (
	"context"
	"fmt"
	"strconv"

	"github.com/pkg/errors"

	"github.com/gofiber/fiber/v2"
	"github.com/initia-labs/opinit-bots-go/executor/batch"
	"github.com/initia-labs/opinit-bots-go/executor/celestia"
	"github.com/initia-labs/opinit-bots-go/executor/child"
	"github.com/initia-labs/opinit-bots-go/executor/host"
	"github.com/initia-labs/opinit-bots-go/server"

	bottypes "github.com/initia-labs/opinit-bots-go/bot/types"
	executortypes "github.com/initia-labs/opinit-bots-go/executor/types"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	"github.com/initia-labs/opinit-bots-go/types"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
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

	homePath string
}

func NewExecutor(cfg *executortypes.Config, db types.DB, sv *server.Server, logger *zap.Logger, homePath string) *Executor {
	err := cfg.Validate()
	if err != nil {
		panic(err)
	}

	executor := &Executor{
		host: host.NewHost(
			cfg.Version, cfg.RelayOracle, cfg.L1NodeConfig(homePath),
			db.WithPrefix([]byte(executortypes.HostNodeName)),
			logger.Named(executortypes.HostNodeName), cfg.L1Bech32Prefix, "",
		),
		child: child.NewChild(
			cfg.Version, cfg.L2NodeConfig(homePath),
			db.WithPrefix([]byte(executortypes.ChildNodeName)),
			logger.Named(executortypes.ChildNodeName), cfg.L2Bech32Prefix,
		),
		batch: batch.NewBatchSubmitter(
			cfg.Version, cfg.L2NodeConfig(homePath),
			cfg.BatchConfig(), db.WithPrefix([]byte(executortypes.BatchNodeName)),
			logger.Named(executortypes.BatchNodeName), cfg.L2ChainID, homePath,
			cfg.DABech32Prefix,
		),

		cfg:    cfg,
		db:     db,
		server: sv,
		logger: logger,

		homePath: homePath,
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

	hostStartHeight, childStartHeight, batchStartHeight, err := executor.getStartHeights()
	if err != nil {
		panic(err)
	}

	err = executor.host.Initialize(hostStartHeight, executor.child, executor.batch, int64(bridgeInfo.BridgeId))
	if err != nil {
		panic(err)
	}
	err = executor.child.Initialize(childStartHeight, executor.host, bridgeInfo)
	if err != nil {
		panic(err)
	}
	err = executor.batch.Initialize(batchStartHeight, executor.host, bridgeInfo)
	if err != nil {
		panic(err)
	}

	da, err := executor.makeDANode(int64(bridgeInfo.BridgeId))
	if err != nil {
		panic(err)
	}
	err = executor.batch.SetDANode(da)
	if err != nil {
		panic(err)
	}

	executor.RegisterQuerier()
	return executor
}

func (ex *Executor) Start(cmdCtx context.Context) error {
	defer ex.Close()
	errGrp, ctx := errgroup.WithContext(cmdCtx)
	ctx = context.WithValue(ctx, types.ContextKeyErrGrp, errGrp)

	errGrp.Go(func() (err error) {
		<-ctx.Done()
		return ex.server.Shutdown()
	})

	errGrp.Go(func() (err error) {
		defer func() {
			ex.logger.Info("api server stopped")
		}()
		return ex.server.Start(ex.cfg.ListenAddress)
	})
	ex.host.Start(ctx)
	ex.child.Start(ctx)
	ex.batch.Start(ctx)
	ex.batch.DA().Start(ctx)
	return errGrp.Wait()
}

func (ex *Executor) Close() {
	ex.db.Close()
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

func (ex *Executor) makeDANode(bridgeId int64) (executortypes.DANode, error) {
	batchInfo := ex.batch.BatchInfo()
	switch batchInfo.BatchInfo.ChainType {
	case ophosttypes.BatchInfo_CHAIN_TYPE_INITIA:
		da := host.NewHost(
			ex.cfg.Version, false, ex.cfg.DANodeConfig(ex.homePath),
			ex.db.WithPrefix([]byte(executortypes.DAHostNodeName)),
			ex.logger.Named(executortypes.DAHostNodeName),
			ex.cfg.DABech32Prefix, batchInfo.BatchInfo.Submitter,
		)
		if ex.host.GetAddress().Equals(da.GetAddress()) {
			return ex.host, nil
		}
		da.SetBridgeId(bridgeId)
		da.RegisterDAHandlers()
		return da, nil
	case ophosttypes.BatchInfo_CHAIN_TYPE_CELESTIA:
		da := celestia.NewDACelestia(ex.cfg.Version, ex.cfg.DANodeConfig(ex.homePath),
			ex.db.WithPrefix([]byte(executortypes.DACelestiaNodeName)),
			ex.logger.Named(executortypes.DACelestiaNodeName),
			ex.cfg.DABech32Prefix, batchInfo.BatchInfo.Submitter,
		)
		err := da.Initialize(ex.batch, bridgeId)
		if err != nil {
			return nil, err
		}
		da.RegisterDAHandlers()
		return da, nil
	}

	return nil, fmt.Errorf("unsupported chain id for DA: %s", ophosttypes.BatchInfo_ChainType_name[int32(batchInfo.BatchInfo.ChainType)])
}

func (ex *Executor) getStartHeights() (l1StartHeight uint64, l2StartHeight uint64, batchStartHeight uint64, err error) {
	if ex.cfg.L2StartHeight != 0 {
		l1StartHeight, l2StartHeight, err = ex.host.QueryHeightsOfOutputTxWithL2BlockNumber(uint64(ex.cfg.L2StartHeight))
	}

	if ex.cfg.BatchStartWithL2Height {
		batchStartHeight = l2StartHeight
	} else {
		batchStartHeight = uint64(ex.cfg.BatchStartHeight)
	}
	return l1StartHeight, l2StartHeight, batchStartHeight, err
}
