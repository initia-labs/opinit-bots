package node

import (
	"context"
	"time"

	comettypes "github.com/cometbft/cometbft/abci/types"
	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	rpccoretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"go.uber.org/zap"
)

type EventHandlerArgs struct {
	BlockHeight     int64
	TxIndex         int64
	EventIndex      int64
	EventAttributes []comettypes.EventAttribute
}

type EventHandlerFn func(EventHandlerArgs) error

const POLLING_INTERVAL = 1 * time.Second
const MSG_QUEUE_SIZE = 100
const GAS_ADJUSTMENT = 1.5
const KEY_NAME = "key"

type Node struct {
	*rpchttp.HTTP

	name   string
	cfg    NodeConfig
	logger *zap.Logger

	lastProcessedHeight int64
	eventHandlers       map[string]EventHandlerFn
	msgQueue            chan sdk.Msg

	cdc        codec.Codec
	txConfig   client.TxConfig
	keyBase    keyring.Keyring
	keyAddress sdk.AccAddress
	txf        tx.Factory
}

func NewNode(name string, cfg NodeConfig, logger *zap.Logger, cdc codec.Codec, txConfig client.TxConfig) (*Node, error) {
	client, err := client.NewClientFromNode(cfg.RPC)

	// Use memory keyring for now
	// TODO: may use os keyring later
	keyBase, err := keyring.New(cfg.ChainID, "memory", "", nil, cdc)
	if err != nil {
		return nil, err
	}

	n := &Node{
		HTTP: client,

		name:          name,
		cfg:           cfg,
		logger:        logger,
		eventHandlers: make(map[string]EventHandlerFn),
		msgQueue:      make(chan sdk.Msg, MSG_QUEUE_SIZE),

		cdc:      cdc,
		txConfig: txConfig,
		keyBase:  keyBase,
	}

	if cfg.Mnemonic != "" {
		_, err := n.keyBase.NewAccount(KEY_NAME, cfg.Mnemonic, "", hd.CreateHDPath(sdk.GetConfig().GetCoinType(), 0, 0).String(), hd.Secp256k1)
		if err != nil {
			return nil, err
		}
		// to check if the key is normally created
		// TODO: delete this code
		key, err := n.keyBase.Key(KEY_NAME)
		if err != nil {
			return nil, err
		}

		addr, err := key.GetAddress()
		if err != nil {
			return nil, err
		}
		n.keyAddress = addr

		n.txf = tx.Factory{}.
			WithAccountRetriever(n).
			WithChainID(cfg.ChainID).
			WithTxConfig(txConfig).
			WithGasAdjustment(GAS_ADJUSTMENT).
			WithGasPrices(cfg.GasPrice).
			WithKeybase(n.keyBase).
			WithSignMode(signing.SignMode_SIGN_MODE_DIRECT)
	}
	return n, err
}

func (n Node) RegisterEventHandler(eventType string, fn EventHandlerFn) {
	n.eventHandlers[eventType] = fn
}

func (n Node) BlockProcessLooper(ctx context.Context) error {
	timer := time.NewTicker(POLLING_INTERVAL)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
		}

		err := n.handleNewBlocks(ctx)
		if err != nil {
			n.logger.Error("failed to get block results", zap.Error(err))
			continue
		}
	}
}

func (n *Node) handleNewBlocks(ctx context.Context) error {
	// TODO: save processed block height & receive new blocks from the last processed block
	status, err := n.Status(ctx)
	if err != nil {
		return err
	}

	latestHeight := status.SyncInfo.LatestBlockHeight
	if n.lastProcessedHeight >= latestHeight {
		return nil
	}

	// TODO: save last processed height and delete this code
	if n.lastProcessedHeight == 0 {
		n.lastProcessedHeight = latestHeight - 1
	}

	for queryHeight := n.lastProcessedHeight + 1; queryHeight <= latestHeight; queryHeight++ {
		blockResult, err := n.BlockResults(ctx, &queryHeight)
		if err != nil {
			return err
		}
		err = n.handleBlockResult(ctx, blockResult)
		if err != nil {
			return err
		}
		n.lastProcessedHeight = queryHeight
	}
	return nil
}

func (n Node) handleBlockResult(ctx context.Context, blockResult *rpccoretypes.ResultBlockResults) error {
	for txIndex, txResult := range blockResult.TxsResults {
		events := txResult.GetEvents()
		for eventIndex, event := range events {
			err := n.handleEvent(blockResult.Height, int64(txIndex), int64(eventIndex), event)
			if err != nil {
				// TODO: handle error
				return err
			}
		}
	}
	return nil
}

func (n Node) handleEvent(height int64, txIndex int64, eventIndex int64, event comettypes.Event) error {
	if n.eventHandlers[event.GetType()] == nil {
		return nil
	}
	n.logger.Info("handle event", zap.String("name", n.name), zap.Int64("height", height), zap.String("type", event.GetType()))

	// Prepare (height, txIndex, eventIndex) to process the event
	err := n.eventHandlers[event.Type](EventHandlerArgs{
		BlockHeight:     height,
		TxIndex:         txIndex,
		EventIndex:      eventIndex,
		EventAttributes: event.GetAttributes(),
	})
	// Store to success event
	return err
}

func (n Node) TxBroadCastLooper(ctx context.Context) {
	// no need to execute this goroutine if mnemonic is not set
	if n.cfg.Mnemonic == "" {
		return
	}
	timer := time.NewTicker(time.Second)
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			// Broadcast tx
			msgs := n.popMsgs()
			if len(msgs) == 0 {
				continue
			}
			err := n.handleTx(ctx, msgs)
			if err != nil {
				n.logger.Error("failed to broadcast tx", zap.String("name", n.name), zap.Error(err))
				continue
			}
		}
	}
}

func (n Node) popMsgs() []sdk.Msg {
	msgs := make([]sdk.Msg, 0, len(n.msgQueue))
	for range len(n.msgQueue) {
		msg := <-n.msgQueue
		msgs = append(msgs, msg)
		// TODO: check total msg size if it is over tx max size or not
		if len(msgs) >= 10 {
			break
		}
	}
	return msgs
}

func (n Node) SendTx(msg sdk.Msg) {
	n.msgQueue <- msg
}
