package node

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"cosmossdk.io/core/address"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/initia-labs/opinit-bots-go/node/broadcaster"
	"github.com/initia-labs/opinit-bots-go/node/rpcclient"
	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"
	"github.com/initia-labs/opinit-bots-go/types"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
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
	lastProcessedBlockHeight uint64
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

	// check if node is catching up
	status, err := rpcClient.Status(context.Background())
	if err != nil {
		return nil, err
	}
	if status.SyncInfo.CatchingUp {
		return nil, errors.New("node is catching up")
	}

	// create broadcaster
	if cfg.BroadcasterConfig != nil {
		n.broadcaster, err = broadcaster.NewBroadcaster(
			*cfg.BroadcasterConfig,
			db,
			logger,
			cdc,
			txConfig,
			rpcClient,
			status,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create broadcaster")
		}
	}

	// load sync info
	err = n.loadSyncInfo()
	if err != nil {
		return nil, err
	}

	return n, nil
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

func (n Node) GetHeight() uint64 {
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
		return nil, errors.New("cannot get broadcaster without broadcaster")
	}

	return n.broadcaster, nil
}

func (n Node) MustGetBroadcaster() *broadcaster.Broadcaster {
	if n.broadcaster == nil {
		panic("cannot get broadcaster without broadcaster")
	}

	return n.broadcaster
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
