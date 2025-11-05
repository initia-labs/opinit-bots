package broadcaster

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"

	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	"github.com/initia-labs/opinit-bots/sentry_integration"
	"github.com/initia-labs/opinit-bots/types"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var ignoringErrors = []error{
	opchildtypes.ErrOracleTimestampNotExists,
	opchildtypes.ErrOracleValidatorsNotRegistered,
	opchildtypes.ErrInvalidOracleHeight,
	opchildtypes.ErrInvalidOracleTimestamp,

	opchildtypes.ErrRedundantTx,
}
var outputIndexRegex = regexp.MustCompile("expected ([0-9]+), got ([0-9]+): invalid output index")

var sentryCapturedErrors = []error{
	sdkerrors.ErrOutOfGas,
	sdkerrors.ErrInsufficientFunds,
}

var ErrAccountSequenceMismatch = errors.New("account sequence mismatch")

// handleMsgError handles error when processing messages.
// If there is an error known to be ignored, it will be ignored.
func (b *Broadcaster) handleMsgError(ctx types.Context, err error, broadcasterAccount *BroadcasterAccount) error {
	expected, got, seqErr := btypes.ParseAccountSequenceMismatch(err.Error())
	if seqErr == nil {
		sentry_integration.CaptureCurrentHubException(err, sentry.LevelWarning)
		if expected > b.peekLastSequenceInLocalPendingTx() {
			broadcasterAccount.UpdateSequence(expected)
		}
		return errors.Wrapf(ErrAccountSequenceMismatch, "expected %d, got %d", expected, got)
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

	for _, e := range sentryCapturedErrors {
		if strings.Contains(err.Error(), e.Error()) {
			sentry_integration.CaptureCurrentHubException(err, sentry.LevelError)
			return err
		}
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

	ctx.Logger().Debug("broadcast tx", zap.String("tx_hash", txHash), zap.Uint64("sequence", sequence))

	if b.cfg.BroadcastOption == btypes.BROADCAST_OPTION_SYNC {
		res, err := b.rpcClient.BroadcastTxSync(ctx, txBytes)
		if err != nil {
			return err
		} else if res.Code != 0 {
			return fmt.Errorf("broadcast txs: %s", res.Log)
		}
	} else {
		res, err := b.rpcClient.CustomBroadcastTxCommit(ctx, txBytes)
		if err != nil {
			return err
		} else if res.TxResult.Code != 0 || res.CheckTx.Code != 0 {
			return fmt.Errorf("broadcast txs: %s, %s", res.TxResult.Log, res.CheckTx.Log)
		}
	}

	stage := b.db.NewStage()
	err = DeleteProcessedMsgs(stage, data)
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
		err = SavePendingTx(stage, pendingTx)
		if err != nil {
			return err
		}
	}

	err = stage.Commit()
	if err != nil {
		return err
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

func (b *Broadcaster) peekLastSequenceInLocalPendingTx() uint64 {
	b.pendingTxMu.Lock()
	defer b.pendingTxMu.Unlock()

	if len(b.pendingTxs) == 0 {
		return 0
	}

	return b.pendingTxs[len(b.pendingTxs)-1].Sequence
}

func (b *Broadcaster) RemovePendingTxsUntil(ctx types.Context, until uint64) error {
	b.pendingTxMu.Lock()
	defer b.pendingTxMu.Unlock()

	start := 0
	stage := b.db.NewStage()

	for _, pendingTx := range b.pendingTxs {
		if pendingTx.Sequence > until {
			break
		}
		start++
		err := DeletePendingTx(stage, pendingTx)
		if err != nil {
			return err
		}
	}

	// apply changes to DB
	if err := stage.Commit(); err != nil {
		return err
	}

	// if successful, remove from local pending txs
	b.pendingTxs = b.pendingTxs[start:]

	return nil
}

func (b *Broadcaster) RebuildPendingTxs(ctx types.Context) (btypes.PendingTxInfo, error) {
	b.pendingTxMu.Lock()
	defer b.pendingTxMu.Unlock()

	if len(b.pendingTxs) == 0 {
		return btypes.PendingTxInfo{}, errors.New("no pending txs")
	}

	newPendingTxs := make([]btypes.PendingTxInfo, len(b.pendingTxs))

	stage := b.db.NewStage()

	err := DeletePendingTxs(stage, b.pendingTxs)
	if err != nil {
		return btypes.PendingTxInfo{}, err
	}

	for i, pendingTx := range b.pendingTxs {

		broadcasterAccount, err := b.AccountByAddress(pendingTx.Sender)
		if err != nil {
			return btypes.PendingTxInfo{}, err
		}
		newTxBytes, newTxHash, err := broadcasterAccount.BuildTxWithNewMemo(ctx, pendingTx.Tx, pendingTx.Sequence)
		if err != nil {
			return btypes.PendingTxInfo{}, err
		}

		newPendingTxs[i] = btypes.PendingTxInfo{
			Sender:          pendingTx.Sender,
			ProcessedHeight: pendingTx.ProcessedHeight,
			Sequence:        pendingTx.Sequence,
			Tx:              newTxBytes,
			TxHash:          newTxHash,
			Timestamp:       pendingTx.Timestamp,
			MsgTypes:        pendingTx.MsgTypes,
			Save:            pendingTx.Save,
		}

		if pendingTx.Save {
			err = SavePendingTx(stage, newPendingTxs[i])
			if err != nil {
				return btypes.PendingTxInfo{}, err
			}
		}
	}

	err = stage.Commit()
	if err != nil {
		return btypes.PendingTxInfo{}, err
	}

	b.pendingTxs = newPendingTxs
	return b.pendingTxs[0], nil
}
