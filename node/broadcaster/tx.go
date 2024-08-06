package broadcaster

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"go.uber.org/zap"

	sdkerrors "cosmossdk.io/errors"
	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"

	btypes "github.com/initia-labs/opinit-bots-go/node/broadcaster/types"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
)

var ignoringErrors = []error{
	opchildtypes.ErrOracleTimestampNotExists,
	opchildtypes.ErrOracleValidatorsNotRegistered,
	opchildtypes.ErrInvalidOracleHeight,
	opchildtypes.ErrInvalidOracleTimestamp,
}
var accountSeqRegex = regexp.MustCompile("account sequence mismatch, expected ([0-9]+), got ([0-9]+)")
var outputIndexRegex = regexp.MustCompile("expected ([0-9]+), got ([0-9]+): invalid output index")

func (b *Broadcaster) handleMsgError(err error) error {
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
			b.txf = b.txf.WithSequence(expected)
		}

		// account sequence mismatched
		// TODO: handle mismatched sequence
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
			b.logger.Warn("ignoring error", zap.String("error", err.Error()))
			return nil
		}

		return err
	}

	for _, e := range ignoringErrors {
		if strings.Contains(err.Error(), e.Error()) {
			b.logger.Warn("ignoring error", zap.String("error", e.Error()))
			return nil
		}
	}

	// b.logger.Error("failed to handle processed msgs", zap.String("error", err.Error()))
	return err
}

// HandleProcessedMsgs handles processed messages by broadcasting them to the network.
// It stores the transaction in the database and local memory and keep track of the successful broadcast.
func (b *Broadcaster) handleProcessedMsgs(ctx context.Context, data btypes.ProcessedMsgs) error {
	sequence := b.txf.Sequence()
	txBytes, txHash, err := b.cfg.BuildTxWithMessages(ctx, data.Msgs)
	if err != nil {
		return sdkerrors.Wrapf(err, "simulation failed")
	}

	res, err := b.rpcClient.BroadcastTxSync(ctx, txBytes)
	if err != nil {
		// TODO: handle error, may repeat sending tx
		return fmt.Errorf("broadcast txs: %w", err)
	}
	if res.Code != 0 {
		return fmt.Errorf("broadcast txs: %s", res.Log)
	}

	b.logger.Debug("broadcast tx", zap.String("tx_hash", txHash), zap.Uint64("sequence", sequence))

	// @sh-cha: maybe we should use data.Save?
	if data.Timestamp != 0 {
		err = b.deleteProcessedMsgs(data.Timestamp)
		if err != nil {
			return err
		}
	}

	b.txf = b.txf.WithSequence(b.txf.Sequence() + 1)
	pendingTx := btypes.PendingTxInfo{
		ProcessedHeight: b.GetHeight(),
		Sequence:        sequence,
		Tx:              txBytes,
		TxHash:          txHash,
		Timestamp:       data.Timestamp,
		Save:            data.Save,
	}

	// save pending transaction to the database for handling after restart
	err = b.savePendingTx(sequence, pendingTx)
	if err != nil {
		return err
	}

	// save pending tx to local memory to handle this tx in this session
	b.enqueueLocalPendingTx(pendingTx)

	return nil
}

// BroadcastTxSync broadcasts transaction bytes to txBroadcastLooper.
func (b Broadcaster) BroadcastMsgs(msgs btypes.ProcessedMsgs) {
	if b.txChannel == nil {
		return
	}

	select {
	case <-b.txChannelStopped:
	case b.txChannel <- msgs:
	}
}

// CalculateGas simulates a tx to generate the appropriate gas settings before broadcasting a tx.
func (b Broadcaster) CalculateGas(ctx context.Context, txf tx.Factory, msgs ...sdk.Msg) (txtypes.SimulateResponse, uint64, error) {
	keyInfo, err := b.keyBase.Key(b.keyName)
	if err != nil {
		return txtypes.SimulateResponse{}, 0, err
	}

	txBytes, err := b.buildSimTx(keyInfo, txf, msgs...)
	if err != nil {
		return txtypes.SimulateResponse{}, 0, err
	}

	simReq := txtypes.SimulateRequest{TxBytes: txBytes}
	reqBytes, err := simReq.Marshal()
	if err != nil {
		return txtypes.SimulateResponse{}, 0, err
	}

	simQuery := abci.RequestQuery{
		Path: "/cosmos.tx.v1beta1.Service/Simulate",
		Data: reqBytes,
	}

	res, err := b.rpcClient.QueryABCI(ctx, simQuery)
	if err != nil {
		return txtypes.SimulateResponse{}, 0, err
	}

	var simRes txtypes.SimulateResponse
	if err := simRes.Unmarshal(res.Value); err != nil {
		return txtypes.SimulateResponse{}, 0, err
	}

	gas, err := b.adjustEstimatedGas(simRes.GasInfo.GasUsed)
	return simRes, gas, err
}

// AdjustEstimatedGas adjusts the estimated gas usage by multiplying it by the gas adjustment factor
// and return estimated gas is higher than max gas error. If the gas usage is zero, the adjusted gas
// is also zero.
func (b Broadcaster) adjustEstimatedGas(gasUsed uint64) (uint64, error) {
	if gasUsed == 0 {
		return gasUsed, nil
	}

	gas := btypes.GAS_ADJUSTMENT * float64(gasUsed)
	if math.IsInf(gas, 1) {
		return 0, fmt.Errorf("infinite gas used")
	}

	return uint64(gas), nil
}

// BuildSimTx creates an unsigned tx with an empty single signature and returns
// the encoded transaction or an error if the unsigned transaction cannot be built.
func (b Broadcaster) buildSimTx(info *keyring.Record, txf tx.Factory, msgs ...sdk.Msg) ([]byte, error) {
	txb, err := txf.BuildUnsignedTx(msgs...)
	if err != nil {
		return nil, err
	}

	var pk cryptotypes.PubKey
	pk, err = info.GetPubKey()
	if err != nil {
		return nil, err
	}

	// Create an empty signature literal as the ante handler will populate with a
	// sentinel pubkey.
	sig := signing.SignatureV2{
		PubKey: pk,
		Data: &signing.SingleSignatureData{
			SignMode: txf.SignMode(),
		},
		Sequence: txf.Sequence(),
	}
	if err := txb.SetSignatures(sig); err != nil {
		return nil, err
	}

	return b.EncodeTx(txb.GetTx())
}

func (b *Broadcaster) enqueueLocalPendingTx(tx btypes.PendingTxInfo) {
	b.pendingTxMu.Lock()
	defer b.pendingTxMu.Unlock()

	b.pendingTxs = append(b.pendingTxs, tx)
}

func (b *Broadcaster) peekLocalPendingTx() btypes.PendingTxInfo {
	b.pendingTxMu.Lock()
	defer b.pendingTxMu.Unlock()

	return b.pendingTxs[0]
}

func (b Broadcaster) lenLocalPendingTx() int {
	b.pendingTxMu.Lock()
	defer b.pendingTxMu.Unlock()

	return len(b.pendingTxs)
}

func (b *Broadcaster) dequeueLocalPendingTx() {
	b.pendingTxMu.Lock()
	defer b.pendingTxMu.Unlock()

	b.pendingTxs = b.pendingTxs[1:]
}

func (b Broadcaster) EncodeTx(tx authsigning.Tx) ([]byte, error) {
	txBytes, err := b.txConfig.TxEncoder()(tx)
	if err != nil {
		return nil, err
	}
	return txBytes, nil
}

func (b Broadcaster) DecodeTx(txBytes []byte) (authsigning.Tx, error) {
	tx, err := b.txConfig.TxDecoder()(txBytes)
	if err != nil {
		return nil, err
	}
	return tx.(authsigning.Tx), nil
}

func (n Broadcaster) ChangeMsgsFromTx(tx authsigning.Tx, msgs []sdk.Msg) (authsigning.Tx, error) {
	builder, err := n.txConfig.WrapTxBuilder(tx)
	if err != nil {
		return nil, err
	}
	err = builder.SetMsgs(msgs...)
	if err != nil {
		return nil, err
	}
	return builder.GetTx(), nil
}

// buildTxWithMessages creates a transaction from the given messages.
func (b Broadcaster) DefaultBuildTxWithMessages(
	ctx context.Context,
	msgs []sdk.Msg,
) (
	txBytes []byte,
	txHash string,
	err error,
) {
	txf := b.txf
	_, adjusted, err := b.CalculateGas(ctx, txf, msgs...)
	if err != nil {
		return nil, "", err
	}

	txf = txf.WithGas(adjusted)
	txb, err := txf.BuildUnsignedTx(msgs...)
	if err != nil {
		return nil, "", err
	}

	if err = tx.Sign(ctx, txf, b.keyName, txb, false); err != nil {
		return nil, "", err
	}

	tx := txb.GetTx()
	txBytes, err = b.EncodeTx(tx)
	if err != nil {
		return nil, "", err
	}
	return txBytes, btypes.TxHash(txBytes), nil
}

func (b Broadcaster) DefaultPendingTxToProcessedMsgs(
	txBytes []byte,
) ([]sdk.Msg, error) {
	tx, err := b.DecodeTx(txBytes)
	if err != nil {
		return nil, err
	}

	return tx.GetMsgs(), nil
}
