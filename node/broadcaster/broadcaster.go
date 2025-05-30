package broadcaster

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"regexp"
	"slices"
	"sync"
	"time"

	"go.uber.org/zap"

	rpccoretypes "github.com/cometbft/cometbft/rpc/core/types"

	"github.com/pkg/errors"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	"github.com/initia-labs/opinit-bots/node/rpcclient"
	"github.com/initia-labs/opinit-bots/types"
)

var txNotFoundRegex = regexp.MustCompile(`tx \(([A-Fa-f0-9]+)\) not found`)

type Broadcaster struct {
	cfg btypes.BroadcasterConfig

	db        types.DB
	cdc       codec.Codec
	rpcClient *rpcclient.RPCClient

	txConfig client.TxConfig
	accounts []*BroadcasterAccount
	// address -> account index
	addressAccountMap map[string]int
	accountMu         *sync.Mutex

	// tx channel to receive processed msgs
	txChannel        chan btypes.ProcessedMsgs
	txChannelStopped chan struct{}

	// local pending txs, which is following Queue data structure
	pendingTxs                []btypes.PendingTxInfo
	pendingProcessedMsgsBatch []btypes.ProcessedMsgs

	pendingTxMu               *sync.Mutex
	broadcastProcessedMsgsMut *sync.Mutex

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

		txConfig:          txConfig,
		accounts:          make([]*BroadcasterAccount, 0),
		addressAccountMap: make(map[string]int),
		accountMu:         &sync.Mutex{},

		txChannel:        make(chan btypes.ProcessedMsgs),
		txChannelStopped: make(chan struct{}),

		pendingTxs:                make([]btypes.PendingTxInfo, 0),
		pendingProcessedMsgsBatch: make([]btypes.ProcessedMsgs, 0),

		pendingTxMu:               &sync.Mutex{},
		broadcastProcessedMsgsMut: &sync.Mutex{},
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

// Initialize initializes the broadcaster with the given keyring configs.
// It loads pending txs and processed msgs batch from the db and prepares the broadcaster.
func (b *Broadcaster) Initialize(ctx types.Context, status *rpccoretypes.ResultStatus, keyringConfigs []btypes.KeyringConfig) error {
	for _, keyringConfig := range keyringConfigs {
		account, err := NewBroadcasterAccount(ctx, b.cfg, b.cdc, b.txConfig, b.rpcClient, keyringConfig)
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

	err := b.prepareBroadcaster(ctx, status.SyncInfo.LatestBlockTime)
	return errors.Wrap(err, "failed to prepare broadcaster")
}

// SetSyncInfo sets the synced height of the broadcaster.
func (b *Broadcaster) SetSyncInfo(height int64) {
	b.syncedHeight = height
}

// prepareBroadcaster prepares the broadcaster by loading pending txs and processed msgs batch from the db.
func (b *Broadcaster) prepareBroadcaster(ctx types.Context, lastBlockTime time.Time) error {
	stage := b.db.NewStage()

	err := b.loadPendingTxs(ctx, stage, lastBlockTime)
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

// loadPendingTxs loads pending txs from db and waits until timeout if there are pending txs.
func (b *Broadcaster) loadPendingTxs(ctx types.Context, stage types.BasicDB, lastBlockTime time.Time) error {
	pendingTxs, err := LoadPendingTxs(b.db)
	if err != nil {
		return err
	}
	ctx.Logger().Debug("load pending txs", zap.Int("count", len(pendingTxs)))

	pollingTimer := time.NewTicker(ctx.PollingInterval())
	defer pollingTimer.Stop()

	reProcessingTxs := make([]btypes.PendingTxInfo, 0)
	retry := 1

	for txIndex := 0; txIndex < len(pendingTxs); {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-pollingTimer.C:
		}

		txHash, err := hex.DecodeString(pendingTxs[txIndex].TxHash)
		if err != nil {
			return fmt.Errorf("failed to decode tx hash; hash: %s, error: %s", pendingTxs[txIndex].TxHash, err.Error())
		}

		res, err := b.rpcClient.QueryTx(ctx, txHash)
		if err == nil && res != nil && res.TxResult.Code == 0 {
			ctx.Logger().Debug("transaction successfully included",
				zap.String("hash", pendingTxs[txIndex].TxHash),
				zap.Int64("height", res.Height))
			txIndex++
		} else if err == nil && res != nil {
			ctx.Logger().Warn("transaction failed",
				zap.String("hash", pendingTxs[txIndex].TxHash),
				zap.Uint32("code", res.TxResult.Code),
				zap.String("log", res.TxResult.Log))
			reProcessingTxs = append(reProcessingTxs, pendingTxs[txIndex])
			txIndex++
		} else if err != nil && txNotFoundRegex.FindStringSubmatch(err.Error()) != nil {
			pendingTxTime := time.Unix(0, pendingTxs[txIndex].Timestamp).UTC()
			timeoutTime := pendingTxTime.Add(b.cfg.TxTimeout)
			if lastBlockTime.After(timeoutTime) {
				reProcessingTxs = append(reProcessingTxs, pendingTxs[txIndex:]...)
				break
			}
		} else {
			ctx.Logger().Warn("retry to query pending tx", zap.String("hash", pendingTxs[txIndex].TxHash), zap.Int("seconds", int(2*math.Exp2(float64(retry)))), zap.Int("count", retry), zap.String("error", err.Error()))
			if types.SleepWithRetry(ctx, retry) {
				return ctx.Err()
			}
			retry++
			if retry > types.MaxRetryCount {
				return fmt.Errorf("failed to query pending tx; hash: %s, error: %s", pendingTxs[txIndex].TxHash, err.Error())
			}
		}
	}

	err = DeletePendingTxs(stage, pendingTxs)
	if err != nil {
		return err
	}

	if len(reProcessingTxs) != 0 {
		processedMsgsBatch, err := b.pendingTxsToProcessedMsgsBatch(ctx, reProcessingTxs)
		if err != nil {
			return err
		}
		b.pendingProcessedMsgsBatch = append(b.pendingProcessedMsgsBatch, processedMsgsBatch...)
	}
	return nil
}

// loadProcessedMsgsBatch loads processed msgs batch from db and updates the timestamp.
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
		processedMsgsBatch[i].Timestamp = types.CurrentNanoTimestamp()
		ctx.Logger().Debug("pending msgs", zap.Int("index", i), zap.String("msgs", processedMsgsBatch[i].String()))
	}

	// save all pending msgs with updated timestamp to db
	b.pendingProcessedMsgsBatch = append(b.pendingProcessedMsgsBatch, processedMsgsBatch...)
	return nil
}

// pendingTxsToProcessedMsgsBatch converts pending txs to processed msgs batch.
func (b *Broadcaster) pendingTxsToProcessedMsgsBatch(ctx types.Context, pendingTxs []btypes.PendingTxInfo) ([]btypes.ProcessedMsgs, error) {
	pendingProcessedMsgsBatch := make([]btypes.ProcessedMsgs, 0)
	// convert pending txs to pending msgs
	for i, pendingTx := range pendingTxs {
		if !pendingTx.Save {
			continue
		}

		account, err := b.AccountByAddress(pendingTx.Sender)
		if err != nil {
			return nil, err
		}
		msgs, err := account.MsgsFromTx(pendingTx.Tx)
		if err != nil {
			return nil, err
		}

		pendingProcessedMsgsBatch = append(pendingProcessedMsgsBatch, btypes.ProcessedMsgs{
			Sender:    pendingTx.Sender,
			Msgs:      msgs,
			Timestamp: types.CurrentNanoTimestamp(),
			Save:      true,
		})
		ctx.Logger().Debug("pending tx", zap.Int("index", i), zap.String("tx", pendingTx.String()))
	}
	return pendingProcessedMsgsBatch, nil
}

// GetHeight returns the current height of the broadcaster.
func (b Broadcaster) GetHeight() int64 {
	return b.syncedHeight + 1
}

// UpdateSyncedHeight updates the synced height of the broadcaster.
func (b *Broadcaster) UpdateSyncedHeight(height int64) {
	b.syncedHeight = height
}

// MsgsToProcessedMsgs converts msgs to processed msgs.
// It splits msgs into chunks of 5 msgs and creates processed msgs for each chunk.
func MsgsToProcessedMsgs(queues map[string][]sdk.Msg) []btypes.ProcessedMsgs {
	res := make([]btypes.ProcessedMsgs, 0)
	for sender := range queues {
		msgs := queues[sender]
		for i := 0; i < len(msgs); i += 5 {
			end := i + 5
			if end > len(msgs) {
				end = len(msgs)
			}

			res = append(res, btypes.ProcessedMsgs{
				Sender:    sender,
				Msgs:      slices.Clone(msgs[i:end]),
				Timestamp: types.CurrentNanoTimestamp(),
				Save:      true,
			})
		}
	}
	return res
}

// AccountByAddress returns the broadcaster account by the given address.
func (b Broadcaster) AccountByAddress(address string) (*BroadcasterAccount, error) {
	b.accountMu.Lock()
	defer b.accountMu.Unlock()
	if _, ok := b.addressAccountMap[address]; !ok {
		return nil, fmt.Errorf("broadcaster account not found; address: %s", address)
	}
	return b.accounts[b.addressAccountMap[address]], nil
}

// AccountByIndex returns the broadcaster account by the given index.
func (b Broadcaster) AccountByIndex(index int) (*BroadcasterAccount, error) {
	b.accountMu.Lock()
	defer b.accountMu.Unlock()
	if len(b.accounts) <= index {
		return nil, fmt.Errorf("broadcaster account not found; length: %d, index: %d", len(b.accounts), index)
	}
	return b.accounts[index], nil
}

func (b Broadcaster) LenProcessedMsgsByMsgType(msgType string) (int, error) {
	processedMsgsBatch, err := LoadProcessedMsgsBatch(b.db, b.cdc)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, processedMsgs := range processedMsgsBatch {
		if slices.Contains(processedMsgs.GetMsgTypes(), msgType) {
			count++
		}
	}
	return count, nil
}

func (b *Broadcaster) LenLocalPendingTxByMsgType(msgType string) (int, error) {
	pendingTxs, err := LoadPendingTxs(b.db)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, tx := range pendingTxs {
		if slices.Contains(tx.MsgTypes, msgType) {
			count++
		}
	}
	return count, nil
}

func (b Broadcaster) BroadcastTxSync(ctx context.Context, txBytes []byte) (*rpccoretypes.ResultBroadcastTx, error) {
	return b.rpcClient.BroadcastTxSync(ctx, txBytes)
}

func (b Broadcaster) BroadcastTxAsync(ctx context.Context, txBytes []byte) (*rpccoretypes.ResultBroadcastTx, error) {
	return b.rpcClient.BroadcastTxAsync(ctx, txBytes)
}
