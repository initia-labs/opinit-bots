package executor

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/initia-labs/opinit-bots/executor/batchsubmitter"
	"github.com/initia-labs/opinit-bots/executor/celestia"
	"github.com/initia-labs/opinit-bots/executor/child"
	"github.com/initia-labs/opinit-bots/executor/host"
	"github.com/initia-labs/opinit-bots/sentry_integration"
	"github.com/initia-labs/opinit-bots/server"
	"github.com/initia-labs/opinit-bots/server/metrics"

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
	host           *host.Host
	child          *child.Child
	batchSubmitter *batchsubmitter.BatchSubmitter

	cfg    *executortypes.Config
	db     types.DB
	server *server.Server
}

func NewExecutor(cfg *executortypes.Config, db types.DB, sv *server.Server) *Executor {
	err := sentry_integration.Init("opinit-bots", string(bottypes.BotTypeExecutor))
	if err != nil {
		panic(err)
	}

	err = cfg.Validate()
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
		batchSubmitter: batchsubmitter.NewBatchSubmitterV1(
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

	childBridgeInfo.BridgeConfig = bridgeInfo.BridgeConfig

	l1StartHeight, l2StartHeight, startOutputIndex, batchStartHeight, err := ex.getNodeStartHeights(ctx, bridgeInfo.BridgeId)
	if err != nil {
		return errors.Wrap(err, "failed to get processed heights")
	}

	hostKeyringConfig, childKeyringConfig, childOracleKeyringConfig, daKeyringConfig := ex.getKeyringConfigs(*bridgeInfo)

	err = ex.host.Initialize(ctx.WithLogger(ctx.Logger().Named("host")), l1StartHeight-1, ex.child, ex.batchSubmitter, *bridgeInfo, hostKeyringConfig)
	if err != nil {
		return errors.Wrap(err, "failed to initialize host")
	}
	err = ex.child.Initialize(ctx.WithLogger(ctx.Logger().Named("child")), l2StartHeight-1, startOutputIndex, ex.host, childBridgeInfo, childKeyringConfig, childOracleKeyringConfig, ex.cfg.DisableDeleteFutureWithdrawal)
	if err != nil {
		return errors.Wrap(err, "failed to initialize child")
	}
	err = ex.batchSubmitter.Initialize(ctx.WithLogger(ctx.Logger().Named("batchSubmitter")), batchStartHeight-1, ex.host, *bridgeInfo)
	if err != nil {
		return errors.Wrap(err, "failed to initialize batch")
	}

	da, err := ex.makeDANode(ctx.WithLogger(ctx.Logger().Named("da")), *bridgeInfo, daKeyringConfig)
	if err != nil {
		return errors.Wrap(err, "failed to make DA node")
	}
	ex.batchSubmitter.SetDANode(da)
	ex.RegisterQuerier(ctx)
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
		return ex.server.Start()
	})

	ctx.ErrGrp().Go(func() error {
		return metrics.StartMetricsUpdater(ctx, ex)
	})

	ex.host.Start(ctx.WithLogger(ctx.Logger().Named("host")))
	ex.child.Start(ctx.WithLogger(ctx.Logger().Named("child")))
	ex.batchSubmitter.Start(ctx.WithLogger(ctx.Logger().Named("batchSubmitter")))
	ex.batchSubmitter.DA().Start(ctx.WithLogger(ctx.Logger().Named("da")))
	return ctx.ErrGrp().Wait()
}

func (ex *Executor) Close() {
	ex.batchSubmitter.Close()
}

// makeDANode creates a DA node based on the bridge info
// - if the bridge chain type is INITIA and the host address is the same as the submitter, it returns the existing host node
// - if the bridge chain type is INITIA and the host address is different from the submitter, it returns a new host node
// - if the bridge chain type is CELESTIA, it returns a new celestia node
func (ex *Executor) makeDANode(ctx types.Context, bridgeInfo ophosttypes.QueryBridgeResponse, daKeyringConfig *btypes.KeyringConfig) (executortypes.DANode, error) {
	if ex.cfg.DisableBatchSubmitter {
		return batchsubmitter.NewNoopDA(), nil
	}

	batchInfo := ex.batchSubmitter.BatchInfo()
	if batchInfo == nil {
		return nil, errors.New("batch info is not set")
	}
	switch batchInfo.BatchInfo.ChainType {
	case ophosttypes.BatchInfo_INITIA:
		// might not exist
		hostAddrStr, err := ex.host.BaseAccountAddressString()
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
	case ophosttypes.BatchInfo_CELESTIA:
		celestiada := celestia.NewDACelestia(
			ex.cfg.Version,
			ex.cfg.DANodeConfig(),
			ex.db.WithPrefix([]byte(types.DAName)),
		)
		err := celestiada.Initialize(ctx, ex.batchSubmitter, bridgeInfo.BridgeId, daKeyringConfig)
		if err != nil {
			return nil, errors.Wrap(err, "failed to initialize celestia DA")
		}
		celestiada.RegisterDAHandlers()
		return celestiada, nil
	}

	return nil, fmt.Errorf("unsupported chain id for DA: %s", ophosttypes.BatchInfo_ChainType_name[int32(batchInfo.BatchInfo.ChainType)])
}

// getNodeStartHeights returns the start heights of the host, the child node, and the batch submitter, and the start output index
func (ex *Executor) getNodeStartHeights(ctx types.Context, bridgeId uint64) (l1StartHeight int64, l2StartHeight int64, startOutputIndex uint64, batchStartHeight int64, err error) {
	var outputL1Height, outputL2Height int64
	var outputIndex uint64

	if ex.host.Node().GetSyncedHeight() == 0 || ex.child.Node().GetSyncedHeight() == 0 {
		// get the last submitted output height before the start height from the host
		output, err := ex.host.QueryOutputByL2BlockNumber(ctx, bridgeId, ex.cfg.L2StartHeight)
		if err != nil {
			return 0, 0, 0, 0, errors.Wrap(err, "failed to query output by l2 block number")
		} else if output != nil {
			outputL1Height = types.MustUint64ToInt64(output.OutputProposal.L1BlockNumber)
			outputL2Height = types.MustUint64ToInt64(output.OutputProposal.L2BlockNumber)
			outputIndex = output.OutputIndex
		}
		l2StartHeight = outputL2Height + 1
		startOutputIndex = outputIndex + 1
	}

	if ex.host.Node().GetSyncedHeight() == 0 {
		// use l1 start height from the config if auto set is disabled
		if ex.cfg.DisableAutoSetL1Height {
			l1StartHeight = ex.cfg.L1StartHeight
		} else {
			childNextL1Sequence, err := ex.child.QueryNextL1Sequence(ctx, 0)
			if err != nil {
				return 0, 0, 0, 0, errors.Wrap(err, "failed to query next l1 sequence")
			}

			// query last NextL1Sequence tx height
			depositTxHeight, err := ex.host.QueryDepositTxHeight(ctx, bridgeId, childNextL1Sequence)
			if err != nil {
				return 0, 0, 0, 0, errors.Wrap(err, "failed to query deposit tx height")
			} else if depositTxHeight == 0 && childNextL1Sequence > 1 {
				// if the deposit tx with next_l1_sequence is not found
				// query deposit tx with next_l1_sequence-1 tx
				depositTxHeight, err = ex.host.QueryDepositTxHeight(ctx, bridgeId, childNextL1Sequence-1)
				if err != nil {
					return 0, 0, 0, 0, errors.Wrap(err, "failed to query deposit tx height")
				}
			}

			if l1StartHeight < depositTxHeight {
				l1StartHeight = depositTxHeight
			}

			if outputL1Height != 0 && outputL1Height+1 < l1StartHeight {
				l1StartHeight = outputL1Height + 1
			}

			// if l1 start height is not set, get the bridge start height from the host
			if l1StartHeight == 0 {
				l1StartHeight, err = ex.host.QueryCreateBridgeHeight(ctx, bridgeId)
				if err != nil {
					return 0, 0, 0, 0, errors.Wrap(err, "failed to query create bridge height")
				}
			}
		}
	}

	if ex.batchSubmitter.Node().GetSyncedHeight() == 0 {
		batchStartHeight = ex.cfg.BatchStartHeight
	}
	return
}

// getKeyringConfigs returns the keyring configs for the host, the child node, the child oracle node, and the DA node
func (ex *Executor) getKeyringConfigs(bridgeInfo ophosttypes.QueryBridgeResponse) (
	hostKeyringConfig *btypes.KeyringConfig,
	childKeyringConfig *btypes.KeyringConfig,
	childOracleKeyringConfig *btypes.KeyringConfig,
	daKeyringConfig *btypes.KeyringConfig,
) {
	if !ex.cfg.DisableOutputSubmitter {
		hostKeyringConfig = &btypes.KeyringConfig{
			Address: bridgeInfo.BridgeConfig.Proposer,
		}
	}

	if ex.cfg.BridgeExecutor != "" {
		childKeyringConfig = &btypes.KeyringConfig{
			Name: ex.cfg.BridgeExecutor,
		}

		if bridgeInfo.BridgeConfig.OracleEnabled && ex.cfg.OracleBridgeExecutor != "" {
			childOracleKeyringConfig = &btypes.KeyringConfig{
				Name:       ex.cfg.OracleBridgeExecutor,
				FeeGranter: childKeyringConfig,
			}
		}
	}

	if !ex.cfg.DisableBatchSubmitter {
		daKeyringConfig = &btypes.KeyringConfig{
			Address: bridgeInfo.BridgeConfig.BatchInfo.Submitter,
		}
	}
	return
}
