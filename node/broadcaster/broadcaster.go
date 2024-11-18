package broadcaster

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"go.uber.org/zap"

	rpccoretypes "github.com/cometbft/cometbft/rpc/core/types"

	"github.com/pkg/errors"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"

	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	"github.com/initia-labs/opinit-bots/node/rpcclient"
	"github.com/initia-labs/opinit-bots/types"
)

type Broadcaster struct {
	cfg btypes.BroadcasterConfig

	db        types.DB
	cdc       codec.Codec
	logger    *zap.Logger
	rpcClient *rpcclient.RPCClient

	txConfig          client.TxConfig
	accounts          []*BroadcasterAccount
	addressAccountMap map[string]int
	accountMu         *sync.Mutex

	txChannel        chan btypes.ProcessedMsgs
	txChannelStopped chan struct{}

	// local pending txs, which is following Queue data structure
	pendingTxMu *sync.Mutex
	pendingTxs  []btypes.PendingTxInfo

	pendingProcessedMsgs []btypes.ProcessedMsgs

	lastProcessedBlockHeight int64
}

func NewBroadcaster(
	cfg btypes.BroadcasterConfig,
	db types.DB,
	logger *zap.Logger,
	cdc codec.Codec,
	txConfig client.TxConfig,
	rpcClient *rpcclient.RPCClient,
) (*Broadcaster, error) {
	b := &Broadcaster{
		cdc:       cdc,
		logger:    logger,
		db:        db,
		rpcClient: rpcClient,

		txConfig:          txConfig,
		accounts:          make([]*BroadcasterAccount, 0),
		addressAccountMap: make(map[string]int),
		accountMu:         &sync.Mutex{},

		txChannel:        make(chan btypes.ProcessedMsgs),
		txChannelStopped: make(chan struct{}),

		pendingTxMu:          &sync.Mutex{},
		pendingTxs:           make([]btypes.PendingTxInfo, 0),
		pendingProcessedMsgs: make([]btypes.ProcessedMsgs, 0),
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

func (b *Broadcaster) Initialize(ctx context.Context, status *rpccoretypes.ResultStatus, keyringConfigs []btypes.KeyringConfig) error {
	for _, keyringConfig := range keyringConfigs {
		account, err := NewBroadcasterAccount(b.cfg, b.cdc, b.txConfig, b.rpcClient, keyringConfig)
		if err != nil {
			return err
		}
		err = account.Load(ctx)
		if err != nil {
			return err
		}
		b.accounts = append(b.accounts, account)
		b.addressAccountMap[account.GetAddressString()] = len(b.accounts) - 1
	}

	// prepare broadcaster
	return b.prepareBroadcaster(ctx, status.SyncInfo.LatestBlockTime)
}

func (b Broadcaster) GetHeight() int64 {
	return b.lastProcessedBlockHeight + 1
}

func (b *Broadcaster) SetSyncInfo(height int64) {
	b.lastProcessedBlockHeight = height
}

func (b *Broadcaster) prepareBroadcaster(ctx context.Context, lastBlockTime time.Time) error {
	dbBatchKVs := make([]types.RawKV, 0)

	loadedPendingTxs, err := b.loadPendingTxs()
	if err != nil {
		return err
	}
	if len(loadedPendingTxs) > 0 {
		pendingTxTime := time.Unix(0, loadedPendingTxs[0].Timestamp)

		// if we have pending txs, wait until timeout
		if timeoutTime := pendingTxTime.Add(b.cfg.TxTimeout); lastBlockTime.Before(timeoutTime) {
			waitingTime := timeoutTime.Sub(lastBlockTime)
			timer := time.NewTimer(waitingTime)
			b.logger.Info("waiting for pending txs to be processed", zap.Duration("waiting_time", waitingTime))
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
			}
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
			account, err := b.AccountByAddress(txInfo.Sender)
			if err != nil {
				return err
			}
			msgs, err := account.PendingTxToProcessedMsgs(txInfo.Tx)
			if err != nil {
				return err
			}

			if txInfo.Save {
				for i := 0; i < len(msgs); i += 5 {
					end := i + 5
					if end > len(msgs) {
						end = len(msgs)
					}

					b.pendingProcessedMsgs = append(b.pendingProcessedMsgs, btypes.ProcessedMsgs{
						Sender:    txInfo.Sender,
						Msgs:      slices.Clone(msgs[i:end]),
						Timestamp: time.Now().UnixNano(),
						Save:      true,
					})
				}
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

func (b Broadcaster) AccountByIndex(index int) (*BroadcasterAccount, error) {
	b.accountMu.Lock()
	defer b.accountMu.Unlock()
	if len(b.accounts) <= index {
		return nil, fmt.Errorf("broadcaster account not found")
	}
	return b.accounts[index], nil
}

func (b Broadcaster) AccountByAddress(address string) (*BroadcasterAccount, error) {
	b.accountMu.Lock()
	defer b.accountMu.Unlock()
	if _, ok := b.addressAccountMap[address]; !ok {
		return nil, fmt.Errorf("broadcaster account not found")
	}
	return b.accounts[b.addressAccountMap[address]], nil
}
