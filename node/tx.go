package node

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	sdkerrors "cosmossdk.io/errors"
	abci "github.com/cometbft/cometbft/abci/types"
	comettypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	nodetypes "github.com/initia-labs/opinit-bots-go/node/types"
	"go.uber.org/zap"

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

func (n *Node) txBroadcastLooper(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case data := <-n.txChannel:
			var err error
			for retry := 1; retry <= 10; retry++ {
				select {
				case <-ctx.Done():
					return nil
				default:
				}
				err = n.handleProcessedMsgs(ctx, data)
				if err == nil {
					break
				} else if err = n.handleMsgError(err); err == nil {
					break
				}
				n.logger.Warn("retry", zap.Int("count", retry), zap.String("error", err.Error()))

				time.Sleep(30 * time.Second)
			}
			if err != nil {
				return errors.Wrap(err, "failed to handle processed msgs")
			}
		}
	}
}

func (n *Node) handleMsgError(err error) error {
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
			n.txf = n.txf.WithSequence(expected)
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
			n.logger.Warn("ignoring error", zap.String("error", err.Error()))
			return nil
		}

		return err
	}

	for _, e := range ignoringErrors {
		if strings.Contains(err.Error(), e.Error()) {
			n.logger.Warn("ignoring error", zap.String("error", e.Error()))
			return nil
		}
	}

	// n.logger.Error("failed to handle processed msgs", zap.String("error", err.Error()))
	return err
}

func (n *Node) handleProcessedMsgs(ctx context.Context, data nodetypes.ProcessedMsgs) error {
	sequence := n.txf.Sequence()
	txBytes, err := n.buildTxWithMessages(n, ctx, data.Msgs)
	if err != nil {
		return sdkerrors.Wrapf(err, "simulation failed")
	}

	res, err := n.BroadcastTxSync(ctx, txBytes)
	if err != nil {
		// TODO: handle error, may repeat sending tx
		return fmt.Errorf("broadcast txs: %w", err)
	}
	if res.Code != 0 {
		return fmt.Errorf("broadcast txs: %s", res.Log)
	}

	n.logger.Debug("broadcast tx", zap.String("tx_hash", TxHash(txBytes)), zap.Uint64("sequence", sequence))

	// @sh-cha: maybe we should use data.Save?
	if data.Timestamp != 0 {
		err = n.deleteProcessedMsgs(data.Timestamp)
		if err != nil {
			return err
		}
	}

	n.txf = n.txf.WithSequence(n.txf.Sequence() + 1)
	pendingTx := nodetypes.PendingTxInfo{
		ProcessedHeight: n.GetHeight(),
		Sequence:        sequence,
		Tx:              txBytes,
		TxHash:          TxHash(txBytes),
		Timestamp:       data.Timestamp,
		Save:            data.Save,
	}

	// save pending transaction to the database for handling after restart
	err = n.savePendingTx(sequence, pendingTx)
	if err != nil {
		return err
	}

	// save pending tx to local memory to handle this tx in this session
	n.enqueueLocalPendingTx(pendingTx)

	return nil
}

// BroadcastTxSync broadcasts transaction bytes to txBroadcastLooper.
func (n *Node) BroadcastMsgs(msgs nodetypes.ProcessedMsgs) {
	if n.txChannel == nil || !n.HasKey() {
		return
	}

	select {
	case <-n.txChannelStopped:
	case n.txChannel <- msgs:
	}
}

// CalculateGas simulates a tx to generate the appropriate gas settings before broadcasting a tx.
func (n *Node) CalculateGas(ctx context.Context, txf tx.Factory, msgs ...sdk.Msg) (txtypes.SimulateResponse, uint64, error) {
	keyInfo, err := n.keyBase.Key(n.keyName)
	if err != nil {
		return txtypes.SimulateResponse{}, 0, err
	}

	txBytes, err := n.buildSimTx(keyInfo, txf, msgs...)
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

	res, err := n.QueryABCI(ctx, simQuery)
	if err != nil {
		return txtypes.SimulateResponse{}, 0, err
	}

	var simRes txtypes.SimulateResponse
	if err := simRes.Unmarshal(res.Value); err != nil {
		return txtypes.SimulateResponse{}, 0, err
	}

	gas, err := n.adjustEstimatedGas(simRes.GasInfo.GasUsed)
	return simRes, gas, err
}

// AdjustEstimatedGas adjusts the estimated gas usage by multiplying it by the gas adjustment factor
// and return estimated gas is higher than max gas error. If the gas usage is zero, the adjusted gas
// is also zero.
func (n *Node) adjustEstimatedGas(gasUsed uint64) (uint64, error) {
	if gasUsed == 0 {
		return gasUsed, nil
	}

	gas := nodetypes.GAS_ADJUSTMENT * float64(gasUsed)
	if math.IsInf(gas, 1) {
		return 0, fmt.Errorf("infinite gas used")
	}
	return uint64(gas), nil
}

// BuildSimTx creates an unsigned tx with an empty single signature and returns
// the encoded transaction or an error if the unsigned transaction cannot be built.
func (n Node) buildSimTx(info *keyring.Record, txf tx.Factory, msgs ...sdk.Msg) ([]byte, error) {
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

	return n.EncodeTx(txb.GetTx())
}

func (n *Node) enqueueLocalPendingTx(tx nodetypes.PendingTxInfo) {
	n.pendingTxMu.Lock()
	defer n.pendingTxMu.Unlock()

	n.pendingTxs = append(n.pendingTxs, tx)
}

func (n *Node) peekLocalPendingTx() nodetypes.PendingTxInfo {
	n.pendingTxMu.Lock()
	defer n.pendingTxMu.Unlock()

	return n.pendingTxs[0]
}

func (n *Node) lenLocalPendingTx() int {
	n.pendingTxMu.Lock()
	defer n.pendingTxMu.Unlock()

	return len(n.pendingTxs)
}

func (n *Node) dequeueLocalPendingTx() {
	n.pendingTxMu.Lock()
	defer n.pendingTxMu.Unlock()

	n.pendingTxs = n.pendingTxs[1:]
}

func (n *Node) EncodeTx(tx authsigning.Tx) ([]byte, error) {
	txBytes, err := n.txConfig.TxEncoder()(tx)
	if err != nil {
		return nil, err
	}
	return txBytes, nil
}

func (n *Node) DecodeTx(txBytes []byte) (authsigning.Tx, error) {
	tx, err := n.txConfig.TxDecoder()(txBytes)
	if err != nil {
		return nil, err
	}
	return tx.(authsigning.Tx), nil
}

func TxHash(txBytes []byte) string {
	return fmt.Sprintf("%X", comettypes.Tx(txBytes).Hash())
}

// buildTxWithMessages creates a transaction from the given messages.
func DefaultBuildTxWithMessages(
	n *Node,
	ctx context.Context,
	msgs []sdk.Msg,
) (
	txBytes []byte,
	err error,
) {
	txf := n.GetTxf()
	_, adjusted, err := n.CalculateGas(ctx, txf, msgs...)
	if err != nil {
		return nil, err
	}

	txf = txf.WithGas(adjusted)
	txb, err := txf.BuildUnsignedTx(msgs...)
	if err != nil {
		return nil, err
	}

	if err = tx.Sign(ctx, txf, n.keyName, txb, false); err != nil {
		return nil, err
	}

	tx := txb.GetTx()
	txBytes, err = n.EncodeTx(tx)
	if err != nil {
		return nil, err
	}
	return txBytes, nil
}

func DefaultPendingTxToProcessedMsgs(
	n *Node,
	txBytes []byte,
) ([]sdk.Msg, error) {
	tx, err := n.DecodeTx(txBytes)
	if err != nil {
		return nil, err
	}
	return tx.GetMsgs(), nil
}
