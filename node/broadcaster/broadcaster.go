package broadcaster

import (
	"slices"
	"sync"
	"time"

	"go.uber.org/zap"

	rpccoretypes "github.com/cometbft/cometbft/rpc/core/types"

	"github.com/pkg/errors"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"

	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	"github.com/initia-labs/opinit-bots/node/rpcclient"
	"github.com/initia-labs/opinit-bots/types"
)

type Broadcaster struct {
	cfg btypes.BroadcasterConfig

	db        types.DB
	cdc       codec.Codec
	rpcClient *rpcclient.RPCClient

	txf      tx.Factory
	txConfig client.TxConfig

	keyBase    keyring.Keyring
	keyName    string
	keyAddress sdk.AccAddress

	txChannel        chan btypes.ProcessedMsgs
	txChannelStopped chan struct{}

	// local pending txs, which is following Queue data structure
	pendingTxMu *sync.Mutex
	pendingTxs  []btypes.PendingTxInfo

	pendingProcessedMsgsBatch []btypes.ProcessedMsgs

	syncedHeight int64
}

func NewBroadcaster(
	cfg btypes.BroadcasterConfig,
	db types.DB,
	cdc codec.Codec,
	txConfig client.TxConfig,
	rpcClient *rpcclient.RPCClient,
) (*Broadcaster, error) {
	b := &Broadcaster{
		cdc:       cdc,
		db:        db,
		rpcClient: rpcClient,

		// txf will be initialized in prepareBroadcaster
		txConfig: txConfig,

		txChannel:        make(chan btypes.ProcessedMsgs),
		txChannelStopped: make(chan struct{}),

		pendingTxMu:               &sync.Mutex{},
		pendingTxs:                make([]btypes.PendingTxInfo, 0),
		pendingProcessedMsgsBatch: make([]btypes.ProcessedMsgs, 0),
	}

	// fill cfg with default functions
	if cfg.MsgsFromTx == nil {
		cfg.WithMsgsFromTxFn(b.DefaultMsgsFromTx)
	}
	if cfg.BuildTxWithMsgs == nil {
		cfg.WithBuildTxWithMsgsFn(b.DefaultBuildTxWithMsgs)
	}

	// validate broadcaster config
	if err := cfg.Validate(); err != nil {
		return nil, errors.Wrap(err, "failed to validate broadcaster config")
	}

	// set config after validation
	b.cfg = cfg

	// validate rpc client
	if rpcClient == nil {
		return nil, errors.New("rpc client is nil")
	}
	return b, nil
}

func (b *Broadcaster) Initialize(ctx types.Context, status *rpccoretypes.ResultStatus, keyringConfig *btypes.KeyringConfig) error {
	err := keyringConfig.Validate()
	if err != nil {
		return err
	}

	// setup keyring
	keyBase, keyringRecord, err := b.cfg.GetKeyringRecord(b.cdc, keyringConfig)
	if err != nil {
		return err
	}
	b.keyBase = keyBase
	addr, err := keyringRecord.GetAddress()
	if err != nil {
		return err
	}
	b.keyAddress = addr
	b.keyName = keyringRecord.Name

	// prepare broadcaster
	return b.prepareBroadcaster(ctx, status.SyncInfo.LatestBlockTime)
}

func (b Broadcaster) getClientCtx() client.Context {
	return client.Context{}.WithClient(b.rpcClient).
		WithInterfaceRegistry(b.cdc.InterfaceRegistry()).
		WithChainID(b.cfg.ChainID).
		WithCodec(b.cdc).
		WithFromAddress(b.keyAddress)
}

func (b Broadcaster) GetTxf() tx.Factory {
	return b.txf
}

func (b *Broadcaster) prepareBroadcaster(ctx types.Context, lastBlockTime time.Time) error {
	b.txf = tx.Factory{}.
		WithAccountRetriever(b).
		WithChainID(b.cfg.ChainID).
		WithTxConfig(b.txConfig).
		WithGasAdjustment(b.cfg.GasAdjustment).
		WithGasPrices(b.cfg.GasPrice).
		WithKeybase(b.keyBase).
		WithSignMode(signing.SignMode_SIGN_MODE_DIRECT)

	err := b.loadAccount()
	if err != nil {
		return err
	}

	stage := b.db.NewStage()

	err = b.loadPendingTxs(ctx, stage, lastBlockTime)
	if err != nil {
		return err
	}

	err = b.loadProcessedMsgsBatch(ctx, stage)
	if err != nil {
		return err
	}

	err = SaveProcessedMsgsBatch(stage, b.cdc, b.pendingProcessedMsgsBatch)
	if err != nil {
		return err
	}

	return stage.Commit()
}

func (b *Broadcaster) loadPendingTxs(ctx types.Context, stage types.BasicDB, lastBlockTime time.Time) error {
	pendingTxs, err := LoadPendingTxs(b.db)
	if err != nil {
		return err
	}
	ctx.Logger().Debug("load pending txs", zap.Int("count", len(pendingTxs)))

	if len(pendingTxs) == 0 {
		return nil
	}

	pendingTxTime := time.Unix(0, pendingTxs[0].Timestamp)
	// if we have pending txs, wait until timeout
	if timeoutTime := pendingTxTime.Add(b.cfg.TxTimeout); lastBlockTime.Before(timeoutTime) {
		waitingTime := timeoutTime.Sub(lastBlockTime)
		timer := time.NewTimer(waitingTime)
		ctx.Logger().Info("waiting for pending txs to be processed", zap.Duration("waiting_time", waitingTime))
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}

	err = DeletePendingTxs(stage, pendingTxs)
	if err != nil {
		return err
	}

	processedMsgsBatch, err := b.pendingTxsToProcessedMsgsBatch(ctx, pendingTxs)
	if err != nil {
		return err
	}
	b.pendingProcessedMsgsBatch = append(b.pendingProcessedMsgsBatch, processedMsgsBatch...)
	return nil
}

func (b *Broadcaster) loadProcessedMsgsBatch(ctx types.Context, stage types.BasicDB) error {
	processedMsgsBatch, err := LoadProcessedMsgsBatch(b.db, b.cdc)
	if err != nil {
		return err
	}
	ctx.Logger().Debug("load pending processed msgs", zap.Int("count", len(processedMsgsBatch)))

	// need to remove processed msgs from db before updating the timestamp
	// because the timestamp is used as a key.
	err = DeleteProcessedMsgsBatch(stage, processedMsgsBatch)
	if err != nil {
		return err
	}

	// update timestamp of loaded processed msgs
	for i := range processedMsgsBatch {
		processedMsgsBatch[i].Timestamp = time.Now().UnixNano()
		ctx.Logger().Debug("pending msgs", zap.Int("index", i), zap.String("msgs", processedMsgsBatch[i].String()))
	}

	// save all pending msgs with updated timestamp to db
	b.pendingProcessedMsgsBatch = append(b.pendingProcessedMsgsBatch, processedMsgsBatch...)
	return nil
}

func (b *Broadcaster) pendingTxsToProcessedMsgsBatch(ctx types.Context, pendingTxs []btypes.PendingTxInfo) ([]btypes.ProcessedMsgs, error) {
	pendingProcessedMsgsBatch := make([]btypes.ProcessedMsgs, 0)

	// convert pending txs to pending msgs
	for i, pendingTx := range pendingTxs {
		if !pendingTx.Save {
			continue
		}

		msgs, err := b.cfg.MsgsFromTx(pendingTx.Tx)
		if err != nil {
			return nil, err
		}

		pendingProcessedMsgsBatch = append(pendingProcessedMsgsBatch, MsgsToProcessedMsgs(msgs)...)
		ctx.Logger().Debug("pending tx", zap.Int("index", i), zap.String("tx", pendingTx.String()))
	}
	return pendingProcessedMsgsBatch, nil
}

func (b Broadcaster) GetHeight() int64 {
	return b.syncedHeight + 1
}

func (b *Broadcaster) UpdateSyncedHeight(height int64) {
	b.syncedHeight = height
}

func (b Broadcaster) KeyName() string {
	return b.keyName
}

func MsgsToProcessedMsgs(msgs []sdk.Msg) []btypes.ProcessedMsgs {
	res := make([]btypes.ProcessedMsgs, 0)
	for i := 0; i < len(msgs); i += 5 {
		end := i + 5
		if end > len(msgs) {
			end = len(msgs)
		}

		res = append(res, btypes.ProcessedMsgs{
			Msgs:      slices.Clone(msgs[i:end]),
			Timestamp: time.Now().UnixNano(),
			Save:      true,
		})
	}
	return res
}
