package txutils

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/initia-labs/OPinit/x/opchild"
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	"github.com/initia-labs/opinit-bots/keys"
	"github.com/stretchr/testify/require"
)

func TestChangeMsgsFromTx(t *testing.T) {
	unlock := keys.SetSDKConfigContext("init")
	_, txConfig, err := keys.CreateCodec([]keys.RegisterInterfaces{
		auth.AppModuleBasic{}.RegisterInterfaces,
		opchild.AppModuleBasic{}.RegisterInterfaces,
	})
	require.NoError(t, err)
	unlock()

	txf := tx.Factory{}.WithChainID("test_chain").WithTxConfig(txConfig)
	txb, err := txf.BuildUnsignedTx(&opchildtypes.MsgUpdateOracle{Sender: "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5"})
	require.NoError(t, err)
	tx := txb.GetTx()
	require.Len(t, tx.GetMsgs(), 1)
	require.Equal(t, sdk.MsgTypeURL(tx.GetMsgs()[0]), "/opinit.opchild.v1.MsgUpdateOracle")
	tx, err = ChangeMsgsFromTx(txConfig, tx, []sdk.Msg{&opchildtypes.MsgFinalizeTokenDeposit{}, &opchildtypes.MsgExecuteMessages{}})
	require.NoError(t, err)
	require.Len(t, tx.GetMsgs(), 2)
	require.Equal(t, sdk.MsgTypeURL(tx.GetMsgs()[0]), "/opinit.opchild.v1.MsgFinalizeTokenDeposit")
	require.Equal(t, sdk.MsgTypeURL(tx.GetMsgs()[1]), "/opinit.opchild.v1.MsgExecuteMessages")
}
