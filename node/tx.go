package node

import (
	"context"
	"fmt"
	"math"

	abci "github.com/cometbft/cometbft/abci/types"
	comettypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"go.uber.org/zap"
)

func (n Node) handleTx(ctx context.Context, msgs []sdk.Msg) error {
	txBytes, err := n.buildMessages(
		ctx,
		msgs,
	)
	if err != nil {
		return err
	}
	hash := fmt.Sprintf("%X", comettypes.Tx(txBytes).Hash())

	n.logger.Info("broadcast tx", zap.String("name", n.name), zap.String("hash", hash), zap.Int("length", len(msgs)))

	// TODO: use sync & wait tx until it is included in a block
	res, err := n.BroadcastTxCommit(ctx, txBytes)
	if err != nil {
		return err
	}

	n.logger.Info("tx result", zap.String("name", n.name), zap.String("hash", hash), zap.Int64("height", res.Height))
	return nil
}

func (n *Node) buildMessages(
	ctx context.Context,
	msgs []sdk.Msg,
) (
	txBytes []byte,
	err error,
) {
	n.txf, err = n.PrepareFactory(n.txf)
	if err != nil {
		return nil, err
	}

	_, adjusted, err := n.CalculateGas(ctx, n.txf, msgs...)
	if err != nil {
		return nil, err
	}

	n.txf = n.txf.WithGas(adjusted)
	// Build the transaction builder
	txb, err := n.txf.BuildUnsignedTx(msgs...)
	if err != nil {
		return nil, err
	}

	if err = tx.Sign(ctx, n.txf, KEY_NAME, txb, false); err != nil {
		return nil, err
	}

	tx := txb.GetTx()
	// Generate the transaction bytes
	txBytes, err = n.txConfig.TxEncoder()(tx)
	if err != nil {
		return nil, err
	}

	return txBytes, nil
}

// PrepareFactory mutates the tx factory with the appropriate account number, sequence number, and min gas settings.
func (n *Node) PrepareFactory(txf tx.Factory) (tx.Factory, error) {
	var (
		err      error
		num, seq uint64
	)

	cliCtx := client.Context{}.WithClient(n).
		WithInterfaceRegistry(n.cdc.InterfaceRegistry()).
		WithChainID(n.cfg.ChainID).
		WithCodec(n.cdc).
		WithFromAddress(n.keyAddress)

	initNum, initSeq := txf.AccountNumber(), txf.Sequence()
	if initNum == 0 || initSeq == 0 {
		num, seq, err = txf.AccountRetriever().GetAccountNumberSequence(cliCtx, n.keyAddress)
		if err != nil {
			return tx.Factory{}, err
		}

		if initNum == 0 {
			txf = txf.WithAccountNumber(num)
		}

		if initSeq == 0 {
			txf = txf.WithSequence(seq)
		}
	}
	return txf, nil
}

// CalculateGas simulates a tx to generate the appropriate gas settings before broadcasting a tx.
func (n *Node) CalculateGas(ctx context.Context, txf tx.Factory, msgs ...sdk.Msg) (txtypes.SimulateResponse, uint64, error) {
	keyInfo, err := n.keyBase.Key(KEY_NAME)
	if err != nil {
		return txtypes.SimulateResponse{}, 0, err
	}

	txBytes, err := BuildSimTx(keyInfo, txf, msgs...)
	if err != nil {
		return txtypes.SimulateResponse{}, 0, err
	}

	simQuery := abci.RequestQuery{
		Path: "/cosmos.tx.v1beta1.Service/Simulate",
		Data: txBytes,
	}

	res, err := n.QueryABCI(ctx, simQuery)
	if err != nil {
		return txtypes.SimulateResponse{}, 0, err
	}

	var simRes txtypes.SimulateResponse
	if err := simRes.Unmarshal(res.Value); err != nil {
		return txtypes.SimulateResponse{}, 0, err
	}

	gas, err := n.AdjustEstimatedGas(simRes.GasInfo.GasUsed)
	return simRes, gas, err
}

// AdjustEstimatedGas adjusts the estimated gas usage by multiplying it by the gas adjustment factor
// and return estimated gas is higher than max gas error. If the gas usage is zero, the adjusted gas
// is also zero.
func (n *Node) AdjustEstimatedGas(gasUsed uint64) (uint64, error) {
	if gasUsed == 0 {
		return gasUsed, nil
	}

	gas := GAS_ADJUSTMENT * float64(gasUsed)
	if math.IsInf(gas, 1) {
		return 0, fmt.Errorf("infinite gas used")
	}
	return uint64(gas), nil
}

// BuildSimTx creates an unsigned tx with an empty single signature and returns
// the encoded transaction or an error if the unsigned transaction cannot be built.
func BuildSimTx(info *keyring.Record, txf tx.Factory, msgs ...sdk.Msg) ([]byte, error) {
	txb, err := txf.BuildUnsignedTx(msgs...)
	if err != nil {
		return nil, err
	}

	pk, err := info.GetPubKey()
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

	protoProvider, ok := txb.(protoTxProvider)
	if !ok {
		return nil, fmt.Errorf("cannot simulate amino tx")
	}

	simReq := txtypes.SimulateRequest{Tx: protoProvider.GetProtoTx()}
	return simReq.Marshal()
}

// protoTxProvider is a type which can provide a proto transaction. It is a
// workaround to get access to the wrapper TxBuilder's method GetProtoTx().
type protoTxProvider interface {
	GetProtoTx() *txtypes.Tx
}
