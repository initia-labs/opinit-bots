package keys

import (
	"testing"

	"github.com/initia-labs/OPinit/x/opchild"
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	"github.com/initia-labs/opinit-bots/txutils"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/x/auth"
)

func TestCreateCodec(t *testing.T) {
	unlock := SetSDKConfigContext("init")
	codec, txConfig, err := CreateCodec([]RegisterInterfaces{
		auth.AppModuleBasic{}.RegisterInterfaces,
		opchild.AppModuleBasic{}.RegisterInterfaces,
	})
	require.NoError(t, err)
	unlock()

	_, _, err = codec.GetMsgV1Signers(&opchildtypes.MsgUpdateOracle{Sender: "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5"})
	require.NoError(t, err)
	_, _, err = codec.GetMsgV1Signers(&opchildtypes.MsgUpdateOracle{Sender: "cosmos1hrasklz3tr6s9rls4r8fjuf0k4zuha6wt4u7jk"})
	require.Error(t, err)

	txf := tx.Factory{}.WithChainID("test_chain").WithTxConfig(txConfig)
	txb, err := txf.BuildUnsignedTx(&opchildtypes.MsgUpdateOracle{})
	require.NoError(t, err)
	txbytes, err := txutils.EncodeTx(txConfig, txb.GetTx())
	require.NoError(t, err)
	_, err = txutils.DecodeTx(txConfig, txbytes)
	require.NoError(t, err)

	unlock = SetSDKConfigContext("cosmos")
	emptyCodec, emptyTxConfig, err := CreateCodec([]RegisterInterfaces{
		auth.AppModuleBasic{}.RegisterInterfaces,
	})
	require.NoError(t, err)
	unlock()

	_, _, err = emptyCodec.GetMsgV1Signers(&opchildtypes.MsgUpdateOracle{})
	require.Error(t, err)

	txf = txf.WithTxConfig(emptyTxConfig)
	txb, err = txf.BuildUnsignedTx(&opchildtypes.MsgUpdateOracle{})
	require.NoError(t, err)
	txbytes, err = txutils.EncodeTx(emptyTxConfig, txb.GetTx())
	require.NoError(t, err)
	_, err = txutils.DecodeTx(emptyTxConfig, txbytes)
	require.Error(t, err)
}
