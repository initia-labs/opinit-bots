package broadcaster

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"github.com/pkg/errors"

	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	"github.com/initia-labs/opinit-bots/types"
)

var ignoringErrors = []error{
	opchildtypes.ErrOracleTimestampNotExists,
	opchildtypes.ErrOracleValidatorsNotRegistered,
	opchildtypes.ErrInvalidOracleHeight,
	opchildtypes.ErrInvalidOracleTimestamp,

	opchildtypes.ErrRedundantTx,
}
var accountSeqRegex = regexp.MustCompile("account sequence mismatch, expected ([0-9]+), got ([0-9]+)")
var outputIndexRegex = regexp.MustCompile("expected ([0-9]+), got ([0-9]+): invalid output index")

// handleMsgError handles error when processing messages.
// If there is an error known to be ignored, it will be ignored.
func (b *Broadcaster) handleMsgError(ctx types.Context, err error, broadcasterAccount *BroadcasterAccount) error {
	if strs := accountSeqRegex.FindStringSubmatch(err.Error()); strs != nil {
		expected, parseErr := strconv.ParseUint(strs[1], 10, 64)
		if parseErr != nil {
			return parseErr
		}
		got, parseErr := strconv.ParseUint(strs[2], 10, 64)
		if parseErr != nil {
			return parseErr
		}

		if expected > got {
			broadcasterAccount.UpdateSequence(expected)
		}
		return err
	}

	if strs := outputIndexRegex.FindStringSubmatch(err.Error()); strs != nil {
		expected, parseErr := strconv.ParseInt(strs[1], 10, 64)
		if parseErr != nil {
			return parseErr
		}
		got, parseErr := strconv.ParseInt(strs[2], 10, 64)
		if parseErr != nil {
			return parseErr
		}

		if expected > got {
			ctx.Logger().Warn("ignoring error", zap.String("error", err.Error()))
			return nil
		}

		return err
	}

	for _, e := range ignoringErrors {
		if strings.Contains(err.Error(), e.Error()) {
			ctx.Logger().Warn("ignoring error", zap.String("error", e.Error()))
			return nil
		}
	}

	// b.logger.Error("failed to handle processed msgs", zap.String("error", err.Error()))
	return err
}

// HandleProcessedMsgs handles processed messages by broadcasting them to the network.
// It stores the transaction in the database and local memory and keep track of the successful broadcast.
func (b *Broadcaster) handleProcessedMsgs(ctx types.Context, data btypes.ProcessedMsgs, broadcasterAccount *BroadcasterAccount) error {
	sequence := broadcasterAccount.Sequence()

	txBytes, txHash, err := broadcasterAccount.BuildTxWithMsgs(ctx, data.Msgs)
	if err != nil {
		return errors.Wrapf(err, "simulation failed")
	}

	res, err := b.rpcClient.BroadcastTxSync(ctx, txBytes)
	if err != nil {
		// TODO: handle error, may repeat sending tx
		return fmt.Errorf("broadcast txs: %w", err)
	}
	if res.Code != 0 {
		return fmt.Errorf("broadcast txs: %s", res.Log)
	}

	ctx.Logger().Debug("broadcast tx", zap.String("tx_hash", txHash), zap.Uint64("sequence", sequence))

	err = DeleteProcessedMsgs(b.db, data)
	if err != nil {
		return err
	}

	broadcasterAccount.IncreaseSequence()
	pendingTx := btypes.PendingTxInfo{
		Sender:          data.Sender,
		ProcessedHeight: b.GetHeight(),
		Sequence:        sequence,
		Tx:              txBytes,
		TxHash:          txHash,
		Timestamp:       data.Timestamp,
		MsgTypes:        data.GetMsgTypes(),
		Save:            data.Save,
	}

	if pendingTx.Save {
		// save pending transaction to the database for handling after restart
		err = SavePendingTx(b.db, pendingTx)
		if err != nil {
			return err
		}
	}

	// save pending tx to local memory to handle this tx in this session
	b.enqueueLocalPendingTx(pendingTx)
	return nil
}

func (b *Broadcaster) enqueueLocalPendingTx(tx btypes.PendingTxInfo) {
	b.pendingTxMu.Lock()
	defer b.pendingTxMu.Unlock()

	b.pendingTxs = append(b.pendingTxs, tx)
}

func (b *Broadcaster) PeekLocalPendingTx() (btypes.PendingTxInfo, error) {
	b.pendingTxMu.Lock()
	defer b.pendingTxMu.Unlock()

	if len(b.pendingTxs) == 0 {
		return btypes.PendingTxInfo{}, errors.New("no pending txs")
	}
	return b.pendingTxs[0], nil
}

func (b Broadcaster) LenLocalPendingTx() int {
	b.pendingTxMu.Lock()
	defer b.pendingTxMu.Unlock()

	return len(b.pendingTxs)
}

func (b *Broadcaster) dequeueLocalPendingTx() {
	b.pendingTxMu.Lock()
	defer b.pendingTxMu.Unlock()

	b.pendingTxs = b.pendingTxs[1:]
}
