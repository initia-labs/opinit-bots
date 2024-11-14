package executor

import (
	"fmt"

	"github.com/pkg/errors"

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
}

func NewExecutor(cfg *executortypes.Config, db types.DB, sv *server.Server) *Executor {
	err := cfg.Validate()
	if err != nil {
		panic(err)
	}

	return &Executor{
		host: host.NewHostV1(
			cfg.L1NodeConfig(),
			db.WithPrefix([]byte(types.HostName)),
		),
		child: child.NewChildV1(
			cfg.L2NodeConfig(),
			db.WithPrefix([]byte(types.ChildName)),
		),
		batch: batch.NewBatchSubmitterV1(
			cfg.L2NodeConfig(),
			cfg.BatchConfig(),
			db.WithPrefix([]byte(types.BatchName)),
			cfg.L2Node.ChainID,
		),

		cfg:    cfg,
		db:     db,
		server: sv,
	}
}

func (ex *Executor) Initialize(ctx types.Context) error {
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

	ctx.Logger().Info(
		"bridge info",
		zap.Uint64("id", bridgeInfo.BridgeId),
		zap.Duration("submission_interval", bridgeInfo.BridgeConfig.SubmissionInterval),
	)

	hostProcessedHeight, childProcessedHeight, processedOutputIndex, batchProcessedHeight, err := ex.getProcessedHeights(ctx, bridgeInfo.BridgeId)
	if err != nil {
		return errors.Wrap(err, "failed to get processed heights")
	}

	hostKeyringConfig, childKeyringConfig, daKeyringConfig := ex.getKeyringConfigs(*bridgeInfo)

	err = ex.host.Initialize(ctx, hostProcessedHeight, ex.child, ex.batch, *bridgeInfo, hostKeyringConfig)
	if err != nil {
		return errors.Wrap(err, "failed to initialize host")
	}
	err = ex.child.Initialize(ctx, childProcessedHeight, processedOutputIndex+1, ex.host, *bridgeInfo, childKeyringConfig)
	if err != nil {
		return errors.Wrap(err, "failed to initialize child")
	}
	err = ex.batch.Initialize(ctx, batchProcessedHeight, ex.host, *bridgeInfo)
	if err != nil {
		return errors.Wrap(err, "failed to initialize batch")
	}

	da, err := ex.makeDANode(ctx, *bridgeInfo, daKeyringConfig)
	if err != nil {
		return errors.Wrap(err, "failed to make DA node")
	}
	ex.batch.SetDANode(da)
	ex.RegisterQuerier()
	return nil
}

func (ex *Executor) Start(ctx types.Context) error {
	defer ex.Close()

	ctx.ErrGrp().Go(func() (err error) {
		<-ctx.Done()
		return ex.server.Shutdown()
	})

	ctx.ErrGrp().Go(func() (err error) {
		defer func() {
			ctx.Logger().Info("api server stopped")
		}()
		return ex.server.Start(ex.cfg.ListenAddress)
	})
	ex.host.Start(ctx)
	ex.child.Start(ctx)
	ex.batch.Start(ctx)
	ex.batch.DA().Start(ctx)
	return ctx.ErrGrp().Wait()
}

func (ex *Executor) Close() {
	ex.batch.Close()
}

func (ex *Executor) makeDANode(ctx types.Context, bridgeInfo ophosttypes.QueryBridgeResponse, daKeyringConfig *btypes.KeyringConfig) (executortypes.DANode, error) {
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
			return nil, errors.Wrap(err, "failed to get host address")
		} else if err == nil && hostAddrStr == batchInfo.BatchInfo.Submitter {
			return ex.host, nil
		}

		hostda := host.NewHostV1(
			ex.cfg.DANodeConfig(),
			ex.db.WithPrefix([]byte(types.DAName)),
		)
		err = hostda.InitializeDA(ctx, bridgeInfo, daKeyringConfig)
		return hostda, errors.Wrap(err, "failed to initialize host DA")
	case ophosttypes.BatchInfo_CHAIN_TYPE_CELESTIA:
		celestiada := celestia.NewDACelestia(
			ex.cfg.Version,
			ex.cfg.DANodeConfig(),
			ex.db.WithPrefix([]byte(types.DAName)),
		)
		err := celestiada.Initialize(ctx, ex.batch, bridgeInfo.BridgeId, daKeyringConfig)
		if err != nil {
			return nil, errors.Wrap(err, "failed to initialize celestia DA")
		}
		celestiada.RegisterDAHandlers()
		return celestiada, nil
	}

	return nil, fmt.Errorf("unsupported chain id for DA: %s", ophosttypes.BatchInfo_ChainType_name[int32(batchInfo.BatchInfo.ChainType)])
}

func (ex *Executor) getProcessedHeights(ctx types.Context, bridgeId uint64) (l1ProcessedHeight int64, l2ProcessedHeight int64, processedOutputIndex uint64, batchProcessedHeight int64, err error) {
	var outputL1BlockNumber int64
	// get the last submitted output height before the start height from the host
	if ex.cfg.L2StartHeight != 0 {
		output, err := ex.host.QueryOutputByL2BlockNumber(ctx, bridgeId, ex.cfg.L2StartHeight)
		if err != nil {
			return 0, 0, 0, 0, errors.Wrap(err, "failed to query output by l2 block number")
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
			return 0, 0, 0, 0, errors.Wrap(err, "failed to query create bridge height")
		}

		l1Sequence, err := ex.child.QueryNextL1Sequence(ctx, 0)
		if err != nil {
			return 0, 0, 0, 0, errors.Wrap(err, "failed to query next l1 sequence")
		}

		// query l1Sequence tx height
		depositTxHeight, err := ex.host.QueryDepositTxHeight(ctx, bridgeId, l1Sequence)
		if err != nil {
			return 0, 0, 0, 0, errors.Wrap(err, "failed to query deposit tx height")
		} else if depositTxHeight == 0 && l1Sequence > 1 {
			// query l1Sequence - 1 tx height
			depositTxHeight, err = ex.host.QueryDepositTxHeight(ctx, bridgeId, l1Sequence-1)
			if err != nil {
				return 0, 0, 0, 0, errors.Wrap(err, "failed to query deposit tx height")
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
	return l1ProcessedHeight, l2ProcessedHeight, processedOutputIndex, batchProcessedHeight, nil
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
