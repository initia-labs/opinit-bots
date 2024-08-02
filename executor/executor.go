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
			cfg.Version, cfg.RelayOracle, cfg.L1NodeConfig(),
			db.WithPrefix([]byte(executortypes.HostNodeName)),
			logger.Named(executortypes.HostNodeName), homePath, "",
		),
		child: child.NewChild(
			cfg.Version, cfg.L2NodeConfig(),
			db.WithPrefix([]byte(executortypes.ChildNodeName)),
			logger.Named(executortypes.ChildNodeName), homePath,
		),
		batch: batch.NewBatchSubmitter(cfg.Version, cfg.L2NodeConfig(), cfg.BatchConfig(), db.WithPrefix([]byte(executortypes.BatchNodeName)), logger.Named(executortypes.BatchNodeName), homePath),

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

	err = executor.host.Initialize(executor.child, executor.batch, int64(bridgeInfo.BridgeId))
	if err != nil {
		panic(err)
	}
	err = executor.child.Initialize(executor.host, bridgeInfo)
	if err != nil {
		panic(err)
	}
	err = executor.batch.Initialize(executor.host, bridgeInfo)
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

func (ex *Executor) Start(cmdCtx context.Context) {
	// err := ex.server.ShutdownWithContext(cmdCtx)
	// if err != nil {
	// 	return err
	// }

	errGrp := cmdCtx.Value("errGrp").(*errgroup.Group)
	if errGrp == nil {
		panic("error group must be set")
	}
	errGrp.Go(func() (err error) {
		return ex.server.Start(ex.cfg.ListenAddress)
	})

	ex.host.Start(cmdCtx)
	ex.child.Start(cmdCtx)
	ex.batch.Start(cmdCtx)
	ex.batch.DA().Start(cmdCtx)
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
			ex.cfg.Version, false, ex.cfg.DANodeConfig(),
			ex.db.WithPrefix([]byte(executortypes.DAHostNodeName)),
			ex.logger.Named(executortypes.DAHostNodeName), ex.homePath, batchInfo.BatchInfo.Submitter,
		)
		if ex.host.GetAddress().Equals(da.GetAddress()) {
			return ex.host, nil
		}
		da.SetBridgeId(bridgeId)
		da.RegisterDAHandlers()
		return da, nil
	case ophosttypes.BatchInfo_CHAIN_TYPE_CELESTIA:
		da := celestia.NewDACelestia(ex.cfg.Version, ex.cfg.DANodeConfig(),
			ex.db.WithPrefix([]byte(executortypes.DACelestiaNodeName)),
			ex.logger.Named(executortypes.DACelestiaNodeName), ex.homePath, batchInfo.BatchInfo.Submitter,
		)
		da.Initialize(ex.batch, bridgeId)
		da.RegisterDAHandlers()
		return da, nil
	}

	return nil, fmt.Errorf("unsupported chain id for DA: %s", ophosttypes.BatchInfo_ChainType_name[int32(batchInfo.BatchInfo.ChainType)])
}
