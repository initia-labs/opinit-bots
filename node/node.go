package node

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"

	"cosmossdk.io/core/address"
	dbtypes "github.com/initia-labs/opinit-bots/db/types"
	"github.com/initia-labs/opinit-bots/node/broadcaster"
	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	"github.com/initia-labs/opinit-bots/node/rpcclient"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"
	"go.uber.org/zap"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
)

type Node struct {
	cfg nodetypes.NodeConfig
	db  types.DB

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
	startHeightInitialized bool
	syncedHeight           int64

	startOnce *sync.Once
	syncing   bool
}

func NewNode(cfg nodetypes.NodeConfig, db types.DB, cdc codec.Codec, txConfig client.TxConfig) (*Node, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	n := &Node{
		cfg: cfg,
		db:  db,

		eventHandlers: make(map[string]nodetypes.EventHandlerFn),

		cdc:      cdc,
		txConfig: txConfig,

		startOnce: &sync.Once{},
		syncing:   true,
	}

	syncedHeight, err := GetSyncInfo(n.db)
	if errors.Is(err, dbtypes.ErrNotFound) {
		syncedHeight = 0
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to load sync info")
	}
	n.UpdateSyncedHeight(syncedHeight)
	return n, nil
}

// StartHeight is the height to start processing.
// If it is 0, the latest height is used.
// If the latest height exists in the database, this is ignored.
func (n *Node) Initialize(ctx types.Context, processedHeight int64, keyringConfig []btypes.KeyringConfig) (err error) {
	// Create RPC client with shared logger from context
	if n.rpcClient == nil {
		n.rpcClient, err = rpcclient.NewRPCClient(n.cdc, n.cfg.RPC, ctx.Logger().Named("rpcclient"))
		if err != nil {
			return errors.Wrap(err, "failed to create RPC client")
		}
	}

	// Create broadcaster if needed
	if n.cfg.BroadcasterConfig != nil && n.broadcaster == nil {
		n.broadcaster, err = broadcaster.NewBroadcaster(
			*n.cfg.BroadcasterConfig,
			n.db,
			n.cdc,
			n.txConfig,
			n.rpcClient,
		)
		if err != nil {
			return errors.Wrap(err, "failed to create broadcaster")
		}
	}

	// check if node is catching up
	status, err := n.rpcClient.Status(ctx)
	if err != nil {
		return err
	}
	if n.HasBroadcaster() {
		err = n.broadcaster.Initialize(ctx, status, keyringConfig)
		if err != nil {
			return err
		}

		// broadcast pending msgs first before executing block process looper
		go n.broadcaster.BroadcastPendingProcessedMsgs()
	}

	// if not found, initialize the height
	if n.GetSyncedHeight() == 0 {
		n.UpdateSyncedHeight(processedHeight)
		n.startHeightInitialized = true
		ctx.Logger().Info("initialize height")
	} else if err != nil {
		return errors.Wrap(err, "failed to load sync info")
	}
	ctx.Logger().Debug("load sync info", zap.Int64("synced_height", n.syncedHeight))
	return nil
}

// HeightInitialized returns true if the start height is initialized.
func (n *Node) HeightInitialized() bool {
	return n.startHeightInitialized
}

func (n *Node) Start(ctx types.Context) {
	n.startOnce.Do(func() {
		n.start(ctx)
	})
}

func (n *Node) start(ctx types.Context) {
	if n.HasBroadcaster() {
		ctx.ErrGrp().Go(func() (err error) {
			defer func() {
				ctx.Logger().Info("tx broadcast looper stopped")
				if r := recover(); r != nil {
					ctx.Logger().Error("tx broadcast looper panic", zap.Any("recover", r))
					err = fmt.Errorf("tx broadcast looper panic: %v", r)
				}
			}()

			return n.broadcaster.Start(ctx)
		})
	}

	enableEventHandler := true
	if n.cfg.ProcessType != nodetypes.PROCESS_TYPE_ONLY_BROADCAST {
		enableEventHandler = false
		ctx.ErrGrp().Go(func() (err error) {
			defer func() {
				ctx.Logger().Info("block process looper stopped")
				if r := recover(); r != nil {
					ctx.Logger().Error("block process looper panic", zap.Any("recover", r))
					err = fmt.Errorf("block process looper panic: %v", r)
				}
			}()

			return n.blockProcessLooper(ctx, n.cfg.ProcessType)
		})
	}

	ctx.ErrGrp().Go(func() (err error) {
		defer func() {
			ctx.Logger().Info("tx checker looper stopped")
			if r := recover(); r != nil {
				ctx.Logger().Error("tx checker panic", zap.Any("recover", r))
				err = fmt.Errorf("tx checker panic: %v", r)
			}
		}()

		return n.txChecker(ctx, enableEventHandler)
	})
}

func (n Node) AccountCodec() address.Codec {
	return n.cdc.InterfaceRegistry().SigningContext().AddressCodec()
}

func (n Node) Codec() codec.Codec {
	return n.cdc
}

func (n Node) DB() types.DB {
	return n.db
}

// GetHeight returns the processing height.
// It returns the synced height + 1.
func (n Node) GetHeight() int64 {
	return n.syncedHeight + 1
}

func (n *Node) UpdateSyncedHeight(height int64) {
	n.syncedHeight = height
	if n.HasBroadcaster() {
		n.broadcaster.UpdateSyncedHeight(height)
	}
}

// GetSyncedHeight returns the synced height.
// It returns the last processed height.
func (n Node) GetSyncedHeight() int64 {
	return n.syncedHeight
}

func (n Node) GetTxConfig() client.TxConfig {
	return n.txConfig
}

func (n Node) HasBroadcaster() bool {
	return n.broadcaster != nil
}

func (n Node) GetBroadcaster() (*broadcaster.Broadcaster, error) {
	if !n.HasBroadcaster() {
		return nil, types.ErrKeyNotSet
	}

	return n.broadcaster, nil
}

func (n Node) MustGetBroadcaster() *broadcaster.Broadcaster {
	if !n.HasBroadcaster() {
		panic("cannot get broadcaster without broadcaster")
	}
	return n.broadcaster
}

func (n Node) GetRPCClient() *rpcclient.RPCClient {
	if n.rpcClient == nil {
		panic("RPC client not initialized - call Initialize() first")
	}
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
