package broadcaster

import (
	"context"
	"fmt"
	"math"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/initia-labs/opinit-bots/keys"
	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	"github.com/initia-labs/opinit-bots/node/rpcclient"
	"github.com/initia-labs/opinit-bots/txutils"

	ctypes "github.com/cometbft/cometbft/rpc/core/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
)

type BroadcasterAccount struct {
	cfg       btypes.BroadcasterConfig
	txf       tx.Factory
	cdc       codec.Codec
	txConfig  client.TxConfig
	rpcClient *rpcclient.RPCClient

	keyName       string
	keyBase       keyring.Keyring
	keyringRecord *keyring.Record
	address       sdk.AccAddress
	addressString string

	BuildTxWithMessages      btypes.BuildTxWithMessagesFn
	PendingTxToProcessedMsgs btypes.PendingTxToProcessedMsgsFn
}

func NewBroadcasterAccount(cfg btypes.BroadcasterConfig, cdc codec.Codec, txConfig client.TxConfig, rpcClient *rpcclient.RPCClient, keyringConfig btypes.KeyringConfig) (*BroadcasterAccount, error) {
	err := keyringConfig.Validate()
	if err != nil {
		return nil, err
	}

	// setup keyring
	keyBase, keyringRecord, err := cfg.GetKeyringRecord(cdc, &keyringConfig)
	if err != nil {
		return nil, err
	}

	addr, err := keyringRecord.GetAddress()
	if err != nil {
		return nil, err
	}

	addrStr, err := keys.EncodeBech32AccAddr(addr, cfg.Bech32Prefix)
	if err != nil {
		return nil, err
	}
	b := &BroadcasterAccount{
		cfg: cfg,

		cdc:       cdc,
		txConfig:  txConfig,
		rpcClient: rpcClient,

		keyName:       keyringRecord.Name,
		keyBase:       keyBase,
		keyringRecord: keyringRecord,
		address:       addr,
		addressString: addrStr,

		BuildTxWithMessages:      keyringConfig.BuildTxWithMessages,
		PendingTxToProcessedMsgs: keyringConfig.PendingTxToProcessedMsgs,
	}

	if b.BuildTxWithMessages == nil {
		b.BuildTxWithMessages = b.DefaultBuildTxWithMessages
	}

	if b.PendingTxToProcessedMsgs == nil {
		b.PendingTxToProcessedMsgs = b.DefaultPendingTxToProcessedMsgs
	}

	b.txf = tx.Factory{}.
		WithAccountRetriever(b).
		WithChainID(cfg.ChainID).
		WithTxConfig(txConfig).
		WithGasAdjustment(cfg.GasAdjustment).
		WithGasPrices(cfg.GasPrice).
		WithKeybase(keyBase).
		WithSignMode(signing.SignMode_SIGN_MODE_DIRECT)

	return b, nil
}

func (b BroadcasterAccount) GetAddress() sdk.AccAddress {
	return b.address
}

func (b BroadcasterAccount) GetAddressString() string {
	return b.addressString
}

func (b BroadcasterAccount) Bech32Prefix() string {
	return b.cfg.Bech32Prefix
}

func (b *BroadcasterAccount) Load(ctx context.Context) error {
	account, err := b.GetAccount(b.getClientCtx(ctx), b.address)
	if err != nil {
		return err
	}
	b.txf = b.txf.WithAccountNumber(account.GetAccountNumber()).WithSequence(account.GetSequence())
	return nil
}

func (b BroadcasterAccount) GetLatestSequence(ctx context.Context) (uint64, error) {
	account, err := b.GetAccount(b.getClientCtx(ctx), b.address)
	if err != nil {
		return 0, err
	}
	return account.GetAccountNumber(), nil
}

func (b BroadcasterAccount) getClientCtx(ctx context.Context) client.Context {
	return client.Context{}.WithClient(b.rpcClient).
		WithInterfaceRegistry(b.cdc.InterfaceRegistry()).
		WithChainID(b.cfg.ChainID).
		WithCodec(b.cdc).
		WithFromAddress(b.address).
		WithCmdContext(ctx)
}

func (b BroadcasterAccount) Sequence() uint64 {
	return b.txf.Sequence()
}

func (b *BroadcasterAccount) IncreaseSequence() {
	b.txf = b.txf.WithSequence(b.txf.Sequence() + 1)
}

func (b *BroadcasterAccount) UpdateSequence(sequence uint64) {
	b.txf = b.txf.WithSequence(sequence)
}

func (b BroadcasterAccount) BroadcastTxSync(ctx context.Context, txBytes []byte) (*ctypes.ResultBroadcastTx, error) {
	return b.rpcClient.BroadcastTxSync(ctx, txBytes)
}

// BuildSimTx creates an unsigned tx with an empty single signature and returns
// the encoded transaction or an error if the unsigned transaction cannot be built.
func (b BroadcasterAccount) buildSimTx(msgs ...sdk.Msg) ([]byte, error) {
	txb, err := b.txf.BuildUnsignedTx(msgs...)
	if err != nil {
		return nil, err
	}

	pk, err := b.keyringRecord.GetPubKey()
	if err != nil {
		return nil, err
	}

	// Create an empty signature literal as the ante handler will populate with a
	// sentinel pubkey.
	sig := signing.SignatureV2{
		PubKey: pk,
		Data: &signing.SingleSignatureData{
			SignMode: b.txf.SignMode(),
		},
		Sequence: b.txf.Sequence(),
	}
	if err := txb.SetSignatures(sig); err != nil {
		return nil, err
	}

	return txutils.EncodeTx(b.txConfig, txb.GetTx())
}

// CalculateGas simulates a tx to generate the appropriate gas settings before broadcasting a tx.
func (b BroadcasterAccount) CalculateGas(ctx context.Context, msgs ...sdk.Msg) (txtypes.SimulateResponse, uint64, error) {
	txBytes, err := b.buildSimTx(msgs...)
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
func (b BroadcasterAccount) adjustEstimatedGas(gasUsed uint64) (uint64, error) {
	if gasUsed == 0 {
		return gasUsed, nil
	}

	gas := b.cfg.GasAdjustment * float64(gasUsed)
	if math.IsInf(gas, 1) {
		return 0, fmt.Errorf("infinite gas used")
	}

	return uint64(gas), nil
}

func (b BroadcasterAccount) SimulateAndSignTx(ctx context.Context, msgs ...sdk.Msg) (authsigning.Tx, error) {
	_, adjusted, err := b.CalculateGas(ctx, msgs...)
	if err != nil {
		return nil, err
	}

	b.txf = b.txf.WithGas(adjusted)
	txb, err := b.txf.BuildUnsignedTx(msgs...)
	if err != nil {
		return nil, err
	}

	if err = tx.Sign(ctx, b.txf, b.keyName, txb, false); err != nil {
		return nil, err
	}
	return txb.GetTx(), nil
}

// buildTxWithMessages creates a transaction from the given messages.
func (b *BroadcasterAccount) DefaultBuildTxWithMessages(
	ctx context.Context,
	msgs []sdk.Msg,
) (
	txBytes []byte,
	txHash string,
	err error,
) {
	tx, err := b.SimulateAndSignTx(ctx, msgs...)
	if err != nil {
		return nil, "", err
	}

	txBytes, err = txutils.EncodeTx(b.txConfig, tx)
	if err != nil {
		return nil, "", err
	}
	return txBytes, btypes.TxHash(txBytes), nil
}

func (b *BroadcasterAccount) DefaultPendingTxToProcessedMsgs(
	txBytes []byte,
) ([]sdk.Msg, error) {
	tx, err := txutils.DecodeTx(b.txConfig, txBytes)
	if err != nil {
		return nil, err
	}

	return tx.GetMsgs(), nil
}
