package node

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"cosmossdk.io/core/address"
	"github.com/initia-labs/opinit-bots/node/broadcaster"
	"github.com/initia-labs/opinit-bots/node/rpcclient"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
)

type Node struct {
	cfg    nodetypes.NodeConfig
	db     types.DB
	logger *zap.Logger

	cdc      codec.Codec
	txConfig client.TxConfig

	rpcClient   *rpcclient.RPCClient
	broadcaster *broadcaster.Broadcaster

	// handlers
	eventHandlers     map[string]nodetypes.EventHandlerFn
	txHandler         nodetypes.TxHandlerFn
	beginBlockHandler nodetypes.BeginBlockHandlerFn
	endBlockHandler   nodetypes.EndBlockHandlerFn
	rawBlockHandler   nodetypes.RawBlockHandlerFn

	// status info
	startHeightInitialized   bool
	lastProcessedBlockHeight int64
	running                  bool
}

func NewNode(cfg nodetypes.NodeConfig, db types.DB, logger *zap.Logger, cdc codec.Codec, txConfig client.TxConfig) (*Node, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	rpcClient, err := rpcclient.NewRPCClient(cdc, cfg.RPC)
	if err != nil {
		return nil, err
	}

	n := &Node{
		rpcClient: rpcClient,

		cfg:    cfg,
		db:     db,
		logger: logger,

		eventHandlers: make(map[string]nodetypes.EventHandlerFn),

		cdc:      cdc,
		txConfig: txConfig,
	}
	// create broadcaster
	if n.cfg.BroadcasterConfig != nil {
		n.broadcaster, err = broadcaster.NewBroadcaster(
			*n.cfg.BroadcasterConfig,
			n.db,
			n.logger,
			n.cdc,
			n.txConfig,
			n.rpcClient,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create broadcaster")
		}
	}

	return n, nil
}

// StartHeight is the height to start processing.
// If it is 0, the latest height is used.
// If the latest height exists in the database, this is ignored.
func (n *Node) Initialize(ctx context.Context, processedHeight int64) (err error) {
	// check if node is catching up
	status, err := n.rpcClient.Status(ctx)
	if err != nil {
		return err
	}
	if status.SyncInfo.CatchingUp {
		return errors.New("node is catching up")
	}
	if n.broadcaster != nil {
		err = n.broadcaster.Initialize(ctx, status)
		if err != nil {
			return err
		}
	}

	// load sync info
	return n.loadSyncInfo(processedHeight)
}

func (n *Node) HeightInitialized() bool {
	return n.startHeightInitialized
}

func (n *Node) Start(ctx context.Context) {
	if n.running {
		return
	}
	n.running = true

	errGrp := ctx.Value(types.ContextKeyErrGrp).(*errgroup.Group)
	if n.broadcaster != nil {
		errGrp.Go(func() (err error) {
			defer func() {
				n.logger.Info("tx broadcast looper stopped")
				if r := recover(); r != nil {
					n.logger.Error("tx broadcast looper panic", zap.Any("recover", r))
					err = fmt.Errorf("tx broadcast looper panic: %v", r)
				}
			}()

			return n.broadcaster.Start(ctx)
		})

		// broadcast pending msgs first before executing block process looper
		n.broadcaster.BroadcastPendingProcessedMsgs()
	}

	if n.cfg.ProcessType == nodetypes.PROCESS_TYPE_ONLY_BROADCAST {
		if n.broadcaster == nil {
			panic("broadcaster cannot be nil with nodetypes.PROCESS_TYPE_ONLY_BROADCAST")
		}

		errGrp.Go(func() (err error) {
			defer func() {
				n.logger.Info("tx checker looper stopped")
				if r := recover(); r != nil {
					n.logger.Error("tx checker panic", zap.Any("recover", r))
					err = fmt.Errorf("tx checker panic: %v", r)
				}
			}()

			return n.txChecker(ctx)
		})
	} else {
		errGrp.Go(func() (err error) {
			defer func() {
				n.logger.Info("block process looper stopped")
				if r := recover(); r != nil {
					n.logger.Error("block process looper panic", zap.Any("recover", r))
					err = fmt.Errorf("block process looper panic: %v", r)
				}
			}()

			return n.blockProcessLooper(ctx, n.cfg.ProcessType)
		})
	}
}

func (n Node) AccountCodec() address.Codec {
	return n.cdc.InterfaceRegistry().SigningContext().AddressCodec()
}

func (n Node) GetHeight() int64 {
	return n.lastProcessedBlockHeight + 1
}

func (n Node) GetTxConfig() client.TxConfig {
	return n.txConfig
}

func (n Node) HasBroadcaster() bool {
	return n.broadcaster != nil
}

func (n Node) GetBroadcaster() (*broadcaster.Broadcaster, error) {
	if n.broadcaster == nil {
		return nil, types.ErrKeyNotSet
	}

	return n.broadcaster, nil
}

func (n Node) MustGetBroadcaster() *broadcaster.Broadcaster {
	if n.broadcaster == nil {
		panic("cannot get broadcaster without broadcaster")
	}

	return n.broadcaster
}

func (n Node) GetStatus() nodetypes.Status {
	s := nodetypes.Status{}
	if n.cfg.ProcessType != nodetypes.PROCESS_TYPE_ONLY_BROADCAST {
		s.LastBlockHeight = n.GetHeight()
	}

	if n.broadcaster != nil {
		s.Broadcaster = &nodetypes.BroadcasterStatus{
			PendingTxs: n.broadcaster.LenLocalPendingTx(),
			Sequence:   n.broadcaster.GetTxf().Sequence(),
		}
	}
	return s
}

func (n Node) GetRPCClient() *rpcclient.RPCClient {
	return n.rpcClient
}

func (n *Node) RegisterTxHandler(fn nodetypes.TxHandlerFn) {
	n.txHandler = fn
}

func (n *Node) RegisterEventHandler(eventType string, fn nodetypes.EventHandlerFn) {
	n.eventHandlers[eventType] = fn
}

func (n *Node) RegisterBeginBlockHandler(fn nodetypes.BeginBlockHandlerFn) {
	n.beginBlockHandler = fn
}

func (n *Node) RegisterEndBlockHandler(fn nodetypes.EndBlockHandlerFn) {
	n.endBlockHandler = fn
}

func (n *Node) RegisterRawBlockHandler(fn nodetypes.RawBlockHandlerFn) {
	n.rawBlockHandler = fn
}
