package executor

import (
	"context"
	"fmt"
	"strconv"

	"github.com/pkg/errors"

	"github.com/gofiber/fiber/v2"
	"github.com/initia-labs/opinit-bots/executor/batch"
	"github.com/initia-labs/opinit-bots/executor/celestia"
	"github.com/initia-labs/opinit-bots/executor/child"
	"github.com/initia-labs/opinit-bots/executor/host"
	"github.com/initia-labs/opinit-bots/server"

	bottypes "github.com/initia-labs/opinit-bots/bot/types"
	executortypes "github.com/initia-labs/opinit-bots/executor/types"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	"github.com/initia-labs/opinit-bots/types"
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

	homePath string
}

func NewExecutor(cfg *executortypes.Config, db types.DB, sv *server.Server, logger *zap.Logger, homePath string) *Executor {
	err := cfg.Validate()
	if err != nil {
		panic(err)
	}

	return &Executor{
		host: host.NewHostV1(
			cfg.L1NodeConfig(homePath),
			db.WithPrefix([]byte(types.HostName)),
			logger.Named(types.HostName), cfg.L1Node.Bech32Prefix, "",
		),
		child: child.NewChildV1(
			cfg.L2NodeConfig(homePath),
			db.WithPrefix([]byte(types.ChildName)),
			logger.Named(types.ChildName), cfg.L2Node.Bech32Prefix,
		),
		batch: batch.NewBatchSubmitterV0(
			cfg.L2NodeConfig(homePath),
			cfg.BatchConfig(), db.WithPrefix([]byte(types.BatchName)),
			logger.Named(types.BatchName), cfg.L2Node.ChainID, homePath,
			cfg.L2Node.Bech32Prefix,
		),

		cfg:    cfg,
		db:     db,
		server: sv,
		logger: logger,

		homePath: homePath,
	}
}

func (ex *Executor) Initialize(ctx context.Context) error {
	bridgeInfo, err := ex.child.QueryBridgeInfo(ctx)
	if err != nil {
		return err
	}
	if bridgeInfo.BridgeId == 0 {
		return errors.New("bridge info is not set")
	}

	ex.logger.Info(
		"bridge info",
		zap.Uint64("id", bridgeInfo.BridgeId),
		zap.Duration("submission_interval", bridgeInfo.BridgeConfig.SubmissionInterval),
	)

	hostStartHeight, childStartHeight, startOutputIndex, batchStartHeight, err := ex.getStartHeights(ctx, bridgeInfo.BridgeId)
	if err != nil {
		return err
	}

	err = ex.host.Initialize(ctx, hostStartHeight, ex.child, ex.batch, bridgeInfo)
	if err != nil {
		return err
	}
	err = ex.child.Initialize(ctx, childStartHeight, startOutputIndex, ex.host, bridgeInfo)
	if err != nil {
		return err
	}
	err = ex.batch.Initialize(ctx, batchStartHeight, ex.host, bridgeInfo)
	if err != nil {
		return err
	}

	da, err := ex.makeDANode(ctx, bridgeInfo)
	if err != nil {
		return err
	}
	err = ex.batch.SetDANode(da)
	if err != nil {
		return err
	}

	ex.RegisterQuerier()
	return nil
}

func (ex *Executor) Start(ctx context.Context) error {
	defer ex.Close()

	errGrp := types.ErrGrp(ctx)
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
	ex.batch.Close()
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
		return c.JSON(ex.GetStatus())
	})
}

func (ex *Executor) makeDANode(ctx context.Context, bridgeInfo opchildtypes.BridgeInfo) (executortypes.DANode, error) {
	batchInfo := ex.batch.BatchInfo()
	switch batchInfo.BatchInfo.ChainType {
	case ophosttypes.BatchInfo_CHAIN_TYPE_INITIA:
		da := host.NewHostV1(
			ex.cfg.DANodeConfig(ex.homePath),
			ex.db.WithPrefix([]byte(types.DAHostName)),
			ex.logger.Named(types.DAHostName),
			ex.cfg.DANode.Bech32Prefix, batchInfo.BatchInfo.Submitter,
		)
		if ex.host.GetAddress().Equals(da.GetAddress()) {
			return ex.host, nil
		}
		err := da.InitializeDA(ctx, bridgeInfo)
		return da, err
	case ophosttypes.BatchInfo_CHAIN_TYPE_CELESTIA:
		da := celestia.NewDACelestia(ex.cfg.Version, ex.cfg.DANodeConfig(ex.homePath),
			ex.db.WithPrefix([]byte(types.DACelestiaName)),
			ex.logger.Named(types.DACelestiaName),
			ex.cfg.DANode.Bech32Prefix, batchInfo.BatchInfo.Submitter,
		)
		err := da.Initialize(ctx, ex.batch, bridgeInfo.BridgeId)
		if err != nil {
			return nil, err
		}
		da.RegisterDAHandlers()
		return da, nil
	}

	return nil, fmt.Errorf("unsupported chain id for DA: %s", ophosttypes.BatchInfo_ChainType_name[int32(batchInfo.BatchInfo.ChainType)])
}

func (ex *Executor) getStartHeights(ctx context.Context, bridgeId uint64) (l1StartHeight uint64, l2StartHeight uint64, startOutputIndex uint64, batchStartHeight uint64, err error) {
	// get the bridge start height from the host
	l1StartHeight, err = ex.host.QueryCreateBridgeHeight(ctx, bridgeId)
	if err != nil {
		return 0, 0, 0, 0, err
	}

	// get the last submitted output height before the start height from the host
	if ex.cfg.L2StartHeight != 0 {
		output, err := ex.host.QueryOutputByL2BlockNumber(ctx, bridgeId, ex.cfg.L2StartHeight)
		if err != nil {
			return 0, 0, 0, 0, err
		} else if output != nil {
			l1StartHeight = output.OutputProposal.L1BlockNumber
			l2StartHeight = output.OutputProposal.L2BlockNumber
			startOutputIndex = output.OutputIndex + 1
		}
	}
	// get the last deposit tx height from the host
	l1Sequence, err := ex.child.QueryNextL1Sequence(ctx, 0)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	if l1Sequence > 1 {
		depositTxHeight, err := ex.host.QueryDepositTxHeight(ctx, bridgeId, l1Sequence-1)
		if err != nil {
			return 0, 0, 0, 0, err
		}
		if l1StartHeight > depositTxHeight {
			l1StartHeight = depositTxHeight
		}
	}

	if l2StartHeight == 0 {
		startOutputIndex = 1
	}
	if ex.cfg.BatchStartHeight > 0 {
		batchStartHeight = ex.cfg.BatchStartHeight - 1
	}
	return l1StartHeight, l2StartHeight, startOutputIndex, batchStartHeight, err
}
