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
	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"

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
			logger.Named(types.HostName),
		),
		child: child.NewChildV1(
			cfg.L2NodeConfig(homePath),
			db.WithPrefix([]byte(types.ChildName)),
			logger.Named(types.ChildName),
		),
		batch: batch.NewBatchSubmitterV1(
			cfg.L2NodeConfig(homePath),
			cfg.BatchConfig(), db.WithPrefix([]byte(types.BatchName)),
			logger.Named(types.BatchName), cfg.L2Node.ChainID, homePath,
		),

		cfg:    cfg,
		db:     db,
		server: sv,
		logger: logger,

		homePath: homePath,
	}
}

func (ex *Executor) Initialize(ctx context.Context) error {
	childBridgeInfo, err := ex.child.QueryBridgeInfo(ctx)
	if err != nil {
		return err
	}
	if childBridgeInfo.BridgeId == 0 {
		return errors.New("bridge info is not set")
	}

	bridgeInfo, err := ex.host.QueryBridgeConfig(ctx, childBridgeInfo.BridgeId)
	if err != nil {
		return err
	}

	ex.logger.Info(
		"bridge info",
		zap.Uint64("id", bridgeInfo.BridgeId),
		zap.Duration("submission_interval", bridgeInfo.BridgeConfig.SubmissionInterval),
	)

	hostProcessedHeight, childProcessedHeight, processedOutputIndex, batchProcessedHeight, err := ex.getProcessedHeights(ctx, bridgeInfo.BridgeId)
	if err != nil {
		return err
	}

	hostKeyringConfig, childKeyringConfig, daKeyringConfig := ex.getKeyringConfigs(*bridgeInfo)

	err = ex.host.Initialize(ctx, hostProcessedHeight, ex.child, ex.batch, *bridgeInfo, hostKeyringConfig)
	if err != nil {
		return err
	}
	err = ex.child.Initialize(ctx, childProcessedHeight, processedOutputIndex+1, ex.host, *bridgeInfo, childKeyringConfig)
	if err != nil {
		return err
	}
	err = ex.batch.Initialize(ctx, batchProcessedHeight, ex.host, *bridgeInfo)
	if err != nil {
		return err
	}

	da, err := ex.makeDANode(ctx, *bridgeInfo, daKeyringConfig)
	if err != nil {
		return err
	}
	ex.batch.SetDANode(da)
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

func (ex *Executor) makeDANode(ctx context.Context, bridgeInfo ophosttypes.QueryBridgeResponse, daKeyringConfig *btypes.KeyringConfig) (executortypes.DANode, error) {
	if ex.cfg.DisableBatchSubmitter {
		return batch.NewNoopDA(), nil
	}

	batchInfo := ex.batch.BatchInfo()
	if batchInfo == nil {
		return nil, errors.New("batch info is not set")
	}
	switch batchInfo.BatchInfo.ChainType {
	case ophosttypes.BatchInfo_CHAIN_TYPE_INITIA:
		// might not exist
		hostAddrStr, err := ex.host.GetAddressStr()
		if err != nil && !errors.Is(err, types.ErrKeyNotSet) {
			return nil, err
		} else if err == nil && hostAddrStr == batchInfo.BatchInfo.Submitter {
			return ex.host, nil
		}

		hostda := host.NewHostV1(
			ex.cfg.DANodeConfig(ex.homePath),
			ex.db.WithPrefix([]byte(types.DAHostName)),
			ex.logger.Named(types.DAHostName),
		)
		err = hostda.InitializeDA(ctx, bridgeInfo, daKeyringConfig)
		return hostda, err
	case ophosttypes.BatchInfo_CHAIN_TYPE_CELESTIA:
		celestiada := celestia.NewDACelestia(ex.cfg.Version, ex.cfg.DANodeConfig(ex.homePath),
			ex.db.WithPrefix([]byte(types.DACelestiaName)),
			ex.logger.Named(types.DACelestiaName),
		)
		err := celestiada.Initialize(ctx, ex.batch, bridgeInfo.BridgeId, daKeyringConfig)
		if err != nil {
			return nil, err
		}
		celestiada.RegisterDAHandlers()
		return celestiada, nil
	}

	return nil, fmt.Errorf("unsupported chain id for DA: %s", ophosttypes.BatchInfo_ChainType_name[int32(batchInfo.BatchInfo.ChainType)])
}

func (ex *Executor) getProcessedHeights(ctx context.Context, bridgeId uint64) (l1ProcessedHeight int64, l2ProcessedHeight int64, processedOutputIndex uint64, batchProcessedHeight int64, err error) {
	var outputL1BlockNumber int64
	// get the last submitted output height before the start height from the host
	if ex.cfg.L2StartHeight != 0 {
		output, err := ex.host.QueryOutputByL2BlockNumber(ctx, bridgeId, ex.cfg.L2StartHeight)
		if err != nil {
			return 0, 0, 0, 0, err
		} else if output != nil {
			outputL1BlockNumber = types.MustUint64ToInt64(output.OutputProposal.L1BlockNumber)
			l2ProcessedHeight = types.MustUint64ToInt64(output.OutputProposal.L2BlockNumber)
			processedOutputIndex = output.OutputIndex
		}
	}

	if ex.cfg.DisableAutoSetL1Height {
		l1ProcessedHeight = ex.cfg.L1StartHeight
	} else {
		// get the bridge start height from the host
		l1ProcessedHeight, err = ex.host.QueryCreateBridgeHeight(ctx, bridgeId)
		if err != nil {
			return 0, 0, 0, 0, err
		}

		l1Sequence, err := ex.child.QueryNextL1Sequence(ctx, 0)
		if err != nil {
			return 0, 0, 0, 0, err
		}

		// query l1Sequence tx height
		depositTxHeight, err := ex.host.QueryDepositTxHeight(ctx, bridgeId, l1Sequence)
		if err != nil {
			return 0, 0, 0, 0, err
		} else if depositTxHeight == 0 && l1Sequence > 1 {
			// query l1Sequence - 1 tx height
			depositTxHeight, err = ex.host.QueryDepositTxHeight(ctx, bridgeId, l1Sequence-1)
			if err != nil {
				return 0, 0, 0, 0, err
			}
		}
		if depositTxHeight > l1ProcessedHeight {
			l1ProcessedHeight = depositTxHeight
		}
		if outputL1BlockNumber != 0 && outputL1BlockNumber < l1ProcessedHeight {
			l1ProcessedHeight = outputL1BlockNumber
		}
	}

	if l1ProcessedHeight > 0 {
		l1ProcessedHeight--
	}

	if ex.cfg.BatchStartHeight > 0 {
		batchProcessedHeight = ex.cfg.BatchStartHeight - 1
	}
	return l1ProcessedHeight, l2ProcessedHeight, processedOutputIndex, batchProcessedHeight, err
}

func (ex *Executor) getKeyringConfigs(bridgeInfo ophosttypes.QueryBridgeResponse) (hostKeyringConfig *btypes.KeyringConfig, childKeyringConfig *btypes.KeyringConfig, daKeyringConfig *btypes.KeyringConfig) {
	if !ex.cfg.DisableOutputSubmitter {
		hostKeyringConfig = &btypes.KeyringConfig{
			Address: bridgeInfo.BridgeConfig.Proposer,
		}
	}

	if ex.cfg.BridgeExecutor != "" {
		childKeyringConfig = &btypes.KeyringConfig{
			Name: ex.cfg.BridgeExecutor,
		}
	}

	if !ex.cfg.DisableBatchSubmitter {
		daKeyringConfig = &btypes.KeyringConfig{
			Address: bridgeInfo.BridgeConfig.BatchInfo.Submitter,
		}
	}
	return
}
