package batchsubmitter

import (
	"testing"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/client/tx"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	"github.com/initia-labs/opinit-bots/db"
	"github.com/initia-labs/opinit-bots/node"
	"github.com/initia-labs/opinit-bots/node/types"
	childprovider "github.com/initia-labs/opinit-bots/provider/child"
	"github.com/initia-labs/opinit-bots/txutils"
	"github.com/stretchr/testify/require"
)

func TestEmptyOracleData(t *testing.T) {
	baseDB, err := db.NewMemDB()
	require.NoError(t, err)

	batchDB := baseDB.WithPrefix([]byte("test_batch"))
	appCodec, txConfig, err := childprovider.GetCodec("init")
	require.NoError(t, err)
	batchNode := node.NewTestNode(types.NodeConfig{}, batchDB, appCodec, txConfig, nil, nil)

	batchSubmitter := BatchSubmitter{node: batchNode}

	createAuthzMsg := func(t *testing.T, sender string, msgs []sdk.Msg) *authz.MsgExec {
		msgsAny := make([]*cdctypes.Any, 0)
		for _, msg := range msgs {
			any, err := cdctypes.NewAnyWithValue(msg)
			require.NoError(t, err)
			msgsAny = append(msgsAny, any)
		}
		return &authz.MsgExec{
			Grantee: sender,
			Msgs:    msgsAny,
		}
	}

	cases := []struct {
		name        string
		txs         [][]sdk.Msg
		expectedTxs [][]sdk.Msg
		err         bool
	}{
		{
			name: "0th oracle tx",
			txs: [][]sdk.Msg{
				{
					&opchildtypes.MsgUpdateOracle{Sender: "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", Data: []byte("oracle_data"), Height: 3},
				},
			},
			expectedTxs: [][]sdk.Msg{
				{
					&opchildtypes.MsgUpdateOracle{Sender: "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", Data: []byte(""), Height: 3},
				},
			},
			err: false,
		},
		{
			name: "0th oracle tx with multiple msgs",
			txs: [][]sdk.Msg{
				{
					&opchildtypes.MsgUpdateOracle{Sender: "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", Data: []byte("oracle_data"), Height: 3},
					&opchildtypes.MsgUpdateOracle{Sender: "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", Data: []byte("oracle_data"), Height: 3},
				},
			},
			expectedTxs: [][]sdk.Msg{
				{
					&opchildtypes.MsgUpdateOracle{Sender: "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", Data: []byte("oracle_data"), Height: 3},
					&opchildtypes.MsgUpdateOracle{Sender: "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", Data: []byte("oracle_data"), Height: 3},
				},
			},
			err: false,
		},
		{
			name: "1st oracle tx",
			txs: [][]sdk.Msg{
				{
					&opchildtypes.MsgFinalizeTokenDeposit{Sender: "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", Data: []byte("token_deposit_data"), Amount: sdk.NewInt64Coin("init", 10), Height: 5},
				},
				{
					&opchildtypes.MsgUpdateOracle{Sender: "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", Data: []byte("oracle_data"), Height: 3},
				},
			},
			expectedTxs: [][]sdk.Msg{
				{
					&opchildtypes.MsgFinalizeTokenDeposit{Sender: "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", Data: []byte("token_deposit_data"), Amount: sdk.NewInt64Coin("init", 10), Height: 5},
				},
				{
					&opchildtypes.MsgUpdateOracle{Sender: "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", Data: []byte("oracle_data"), Height: 3},
				},
			},
			err: false,
		},
		{
			name:        "no txs",
			txs:         [][]sdk.Msg{},
			expectedTxs: [][]sdk.Msg{},
			err:         false,
		},
		{
			name: "oracle authz tx",
			txs: [][]sdk.Msg{
				{
					createAuthzMsg(t, "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", []sdk.Msg{&opchildtypes.MsgUpdateOracle{Sender: "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", Data: []byte("oracle_data"), Height: 3}}),
				},
			},
			expectedTxs: [][]sdk.Msg{
				{
					createAuthzMsg(t, "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", []sdk.Msg{&opchildtypes.MsgUpdateOracle{Sender: "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", Data: []byte(""), Height: 3}}),
				},
			},
			err: false,
		},
		{
			name: "authz tx with another msg",
			txs: [][]sdk.Msg{
				{
					createAuthzMsg(t, "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", []sdk.Msg{&opchildtypes.MsgFinalizeTokenDeposit{Sender: "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", Data: []byte("token_deposit_data"), Amount: sdk.NewInt64Coin("init", 10), Height: 5}}),
				},
			},
			expectedTxs: [][]sdk.Msg{
				{
					createAuthzMsg(t, "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", []sdk.Msg{&opchildtypes.MsgFinalizeTokenDeposit{Sender: "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", Data: []byte("token_deposit_data"), Amount: sdk.NewInt64Coin("init", 10), Height: 5}}),
				},
			},
			err: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			txf := tx.Factory{}.WithChainID("test_chain").WithTxConfig(txConfig)
			pbb := cmtproto.Block{
				Data: cmtproto.Data{
					Txs: [][]byte{},
				},
			}

			for _, msgs := range tc.txs {
				txb, err := txf.BuildUnsignedTx(msgs...)
				require.NoError(t, err)
				txBytes, err := txutils.EncodeTx(txConfig, txb.GetTx())
				require.NoError(t, err)
				pbb.Data.Txs = append(pbb.Data.Txs, txBytes)
			}

			changedBlock, err := batchSubmitter.emptyOracleData(&pbb)
			require.NoError(t, err)

			changedBlockTxs := changedBlock.Data.GetTxs()
			require.Len(t, changedBlockTxs, len(tc.expectedTxs))

			for i, txBytes := range changedBlockTxs {
				tx, err := txutils.DecodeTx(txConfig, txBytes)
				require.NoError(t, err)
				for j, actual := range tx.GetMsgs() {
					require.Equal(t, tc.expectedTxs[i][j].String(), actual.String())
				}
			}
		})
	}
}
