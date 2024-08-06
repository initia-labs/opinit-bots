package broadcaster

import (
	"sync"
	"time"

	"go.uber.org/zap"

	rpccoretypes "github.com/cometbft/cometbft/rpc/core/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/pkg/errors"

	btypes "github.com/initia-labs/opinit-bots-go/node/broadcaster/types"
	"github.com/initia-labs/opinit-bots-go/node/rpcclient"
	"github.com/initia-labs/opinit-bots-go/types"
)

type Broadcaster struct {
	cfg btypes.BroadcasterConfig

	db        types.DB
	cdc       codec.Codec
	logger    *zap.Logger
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

	pendingProcessedMsgs []btypes.ProcessedMsgs

	lastProcessedBlockHeight uint64
}

func NewBroadcaster(
	cfg btypes.BroadcasterConfig,
	db types.DB,
	logger *zap.Logger,
	cdc codec.Codec,
	txConfig client.TxConfig,
	rpcClient *rpcclient.RPCClient,
	status *rpccoretypes.ResultStatus,
) (*Broadcaster, error) {
	b := &Broadcaster{
		cdc:       cdc,
		logger:    logger,
		db:        db,
		rpcClient: rpcClient,

		// txf will be initialized in prepareBroadcaster
		txConfig: txConfig,

		txChannel:        make(chan btypes.ProcessedMsgs),
		txChannelStopped: make(chan struct{}),

		pendingTxMu:          &sync.Mutex{},
		pendingTxs:           make([]btypes.PendingTxInfo, 0),
		pendingProcessedMsgs: make([]btypes.ProcessedMsgs, 0),
	}

	// fill cfg with default functions
	if cfg.PendingTxToProcessedMsgs == nil {
		cfg.WithPendingTxToProcessedMsgsFn(b.DefaultPendingTxToProcessedMsgs)
	}
	if cfg.BuildTxWithMessages == nil {
		cfg.WithBuildTxWithMessagesFn(b.DefaultBuildTxWithMessages)
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

	// setup keyring
	keyBase, keyringRecord, err := cfg.GetKeyringRecord(cdc, cfg.ChainID)
	if err != nil {
		return nil, err
	}
	b.keyBase = keyBase
	addr, err := keyringRecord.GetAddress()
	if err != nil {
		return nil, err
	}
	b.keyAddress = addr
	b.keyName = keyringRecord.Name

	// prepare broadcaster
	err = b.prepareBroadcaster(uint64(status.SyncInfo.LatestBlockHeight), status.SyncInfo.LatestBlockTime)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (b Broadcaster) getClientCtx() client.Context {
	return client.Context{}.WithClient(b.rpcClient).
		WithInterfaceRegistry(b.cdc.InterfaceRegistry()).
		WithChainID(b.cfg.ChainID).
		WithCodec(b.cdc).
		WithFromAddress(b.keyAddress)
}

func (n Broadcaster) GetTxf() tx.Factory {
	return n.txf
}

func (b *Broadcaster) prepareBroadcaster(_ /*lastBlockHeight*/ uint64, lastBlockTime time.Time) error {
	b.txf = tx.Factory{}.
		WithAccountRetriever(b).
		WithChainID(b.cfg.ChainID).
		WithTxConfig(b.txConfig).
		WithGasAdjustment(btypes.GAS_ADJUSTMENT).
		WithGasPrices(b.cfg.GasPrice).
		WithKeybase(b.keyBase).
		WithSignMode(signing.SignMode_SIGN_MODE_DIRECT)

	err := b.loadAccount()
	if err != nil {
		return err
	}

	dbBatchKVs := make([]types.RawKV, 0)

	loadedPendingTxs, err := b.loadPendingTxs()
	if err != nil {
		return err
	}

	// TODO: handle mismatched sequence & pending txs
	if len(loadedPendingTxs) > 0 {
		pendingTxTime := time.Unix(0, loadedPendingTxs[0].Timestamp)

		// if we have pending txs, wait until timeout
		if timeoutTime := pendingTxTime.Add(btypes.TX_TIMEOUT); lastBlockTime.Before(timeoutTime) {
			timer := time.NewTimer(timeoutTime.Sub(lastBlockTime))
			<-timer.C
		}

		// convert pending txs to raw kv pairs for deletion
		pendingKVs, err := b.PendingTxsToRawKV(loadedPendingTxs, true)
		if err != nil {
			return err
		}

		// add pending txs delegation to db batch
		dbBatchKVs = append(dbBatchKVs, pendingKVs...)

		// convert pending txs to pending msgs
		for i, txInfo := range loadedPendingTxs {
			msgs, err := b.cfg.PendingTxToProcessedMsgs(txInfo.Tx)
			if err != nil {
				return err
			}

			if txInfo.Save {
				b.pendingProcessedMsgs = append(b.pendingProcessedMsgs, btypes.ProcessedMsgs{
					Msgs:      msgs,
					Timestamp: time.Now().UnixNano(),
					Save:      txInfo.Save,
				})
			}

			b.logger.Debug("pending tx", zap.Int("index", i), zap.String("tx", txInfo.String()))
		}
	}

	loadedProcessedMsgs, err := b.loadProcessedMsgs()
	if err != nil {
		return err
	}

	// need to remove processed msgs from db before updating the timestamp
	// because the timestamp is used as a key.
	kvProcessedMsgs, err := b.ProcessedMsgsToRawKV(loadedProcessedMsgs, true)
	if err != nil {
		return err
	}
	dbBatchKVs = append(dbBatchKVs, kvProcessedMsgs...)

	// update timestamp of loaded processed msgs
	for i, pendingMsgs := range loadedProcessedMsgs {
		loadedProcessedMsgs[i].Timestamp = time.Now().UnixNano()
		b.logger.Debug("pending msgs", zap.Int("index", i), zap.String("msgs", pendingMsgs.String()))
	}

	// save all pending msgs with updated timestamp to db
	b.pendingProcessedMsgs = append(b.pendingProcessedMsgs, loadedProcessedMsgs...)
	kvProcessedMsgs, err = b.ProcessedMsgsToRawKV(b.pendingProcessedMsgs, false)
	if err != nil {
		return err
	}
	dbBatchKVs = append(dbBatchKVs, kvProcessedMsgs...)

	// save all pending msgs first, then broadcast them
	err = b.db.RawBatchSet(dbBatchKVs...)
	if err != nil {
		return err
	}

	return nil
}

func (b *Broadcaster) SetSyncInfo(height uint64) {
	b.lastProcessedBlockHeight = height
}

func (b Broadcaster) KeyName() string {
	return b.keyName
}
