package txutils

import (
	"fmt"

	comettypes "github.com/cometbft/cometbft/types"

	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
)

func EncodeTx(txConfig client.TxConfig, tx authsigning.Tx) ([]byte, error) {
	txBytes, err := txConfig.TxEncoder()(tx)
	if err != nil {
		return nil, err
	}

	return txBytes, nil
}

func DecodeTx(txConfig client.TxConfig, txBytes []byte) (authsigning.Tx, error) {
	tx, err := txConfig.TxDecoder()(txBytes)
	if err != nil {
		return nil, err
	}

	return tx.(authsigning.Tx), nil
}

func ChangeMsgsFromTx(txConfig client.TxConfig, tx authsigning.Tx, msgs []sdk.Msg) (authsigning.Tx, error) {
	builder, err := txConfig.WrapTxBuilder(tx)
	if err != nil {
		return nil, err
	}

	err = builder.SetMsgs(msgs...)
	if err != nil {
		return nil, err
	}

	return builder.GetTx(), nil
}

func TxHash(txBytes []byte) string {
	return fmt.Sprintf("%X", comettypes.Tx(txBytes).Hash())
}
