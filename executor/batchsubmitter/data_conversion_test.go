package batchsubmitter

import (
	"context"
	"testing"
	"time"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttypes "github.com/cometbft/cometbft/types"
	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	"github.com/initia-labs/opinit-bots/db"
	"github.com/initia-labs/opinit-bots/node"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	childprovider "github.com/initia-labs/opinit-bots/provider/child"
	"github.com/initia-labs/opinit-bots/txutils"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	ibcclienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ibctmlightclients "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"

	"github.com/cosmos/cosmos-sdk/client/tx"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
)

func TestEmptyOracleData(t *testing.T) {
	baseDB, err := db.NewMemDB()
	require.NoError(t, err)

	batchDB := baseDB.WithPrefix([]byte("test_batch"))
	appCodec, txConfig, err := childprovider.GetCodec("init")
	require.NoError(t, err)
	batchNode := node.NewTestNode(nodetypes.NodeConfig{}, batchDB, appCodec, txConfig, nil, nil)

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

func TestEmptyRelayOracleData(t *testing.T) {
	baseDB, err := db.NewMemDB()
	require.NoError(t, err)

	batchDB := baseDB.WithPrefix([]byte("test_batch"))
	appCodec, txConfig, err := childprovider.GetCodec("init")
	require.NoError(t, err)
	batchNode := node.NewTestNode(nodetypes.NodeConfig{}, batchDB, appCodec, txConfig, nil, nil)

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
			name: "relay oracle data with prices and proof",
			txs: [][]sdk.Msg{
				{
					&opchildtypes.MsgRelayOracleData{
						Sender: "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5",
						OracleData: opchildtypes.OracleData{
							BridgeId:        1,
							OraclePriceHash: []byte("price_hash"),
							Prices: []opchildtypes.OraclePriceData{
								{CurrencyPair: "BTC/USD", Price: "50000.00", Decimals: 8, Nonce: 1},
								{CurrencyPair: "ETH/USD", Price: "3000.00", Decimals: 8, Nonce: 1},
							},
							L1BlockHeight: 100,
							L1BlockTime:   1000000000,
							Proof:         []byte("merkle_proof_data"),
							ProofHeight:   ibcclienttypes.NewHeight(1, 101),
						},
					},
				},
			},
			expectedTxs: [][]sdk.Msg{
				{
					&opchildtypes.MsgRelayOracleData{
						Sender: "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5",
						OracleData: opchildtypes.OracleData{
							BridgeId:        1,
							OraclePriceHash: []byte("price_hash"),
							Prices:          []opchildtypes.OraclePriceData{},
							L1BlockHeight:   100,
							L1BlockTime:     1000000000,
							Proof:           []byte{},
							ProofHeight:     ibcclienttypes.NewHeight(1, 101),
						},
					},
				},
			},
			err: false,
		},
		{
			name: "relay oracle data in authz wrapper",
			txs: [][]sdk.Msg{
				{
					createAuthzMsg(t, "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", []sdk.Msg{
						&opchildtypes.MsgRelayOracleData{
							Sender: "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5",
							OracleData: opchildtypes.OracleData{
								BridgeId:        1,
								OraclePriceHash: []byte("price_hash"),
								Prices: []opchildtypes.OraclePriceData{
									{CurrencyPair: "BTC/USD", Price: "50000.00", Decimals: 8, Nonce: 1},
								},
								L1BlockHeight: 100,
								L1BlockTime:   1000000000,
								Proof:         []byte("merkle_proof_data"),
								ProofHeight:   ibcclienttypes.NewHeight(1, 101),
							},
						},
					}),
				},
			},
			expectedTxs: [][]sdk.Msg{
				{
					createAuthzMsg(t, "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", []sdk.Msg{
						&opchildtypes.MsgRelayOracleData{
							Sender: "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5",
							OracleData: opchildtypes.OracleData{
								BridgeId:        1,
								OraclePriceHash: []byte("price_hash"),
								Prices:          []opchildtypes.OraclePriceData{},
								L1BlockHeight:   100,
								L1BlockTime:     1000000000,
								Proof:           []byte{},
								ProofHeight:     ibcclienttypes.NewHeight(1, 101),
							},
						},
					}),
				},
			},
			err: false,
		},
		{
			name: "mixed txs - only relay oracle data modified",
			txs: [][]sdk.Msg{
				{
					&opchildtypes.MsgFinalizeTokenDeposit{Sender: "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", Data: []byte("token_deposit_data"), Amount: sdk.NewInt64Coin("init", 10), Height: 5},
				},
				{
					&opchildtypes.MsgRelayOracleData{
						Sender: "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5",
						OracleData: opchildtypes.OracleData{
							BridgeId:        1,
							OraclePriceHash: []byte("price_hash"),
							Prices: []opchildtypes.OraclePriceData{
								{CurrencyPair: "BTC/USD", Price: "50000.00", Decimals: 8, Nonce: 1},
							},
							L1BlockHeight: 100,
							L1BlockTime:   1000000000,
							Proof:         []byte("merkle_proof_data"),
							ProofHeight:   ibcclienttypes.NewHeight(1, 101),
						},
					},
				},
			},
			expectedTxs: [][]sdk.Msg{
				{
					&opchildtypes.MsgFinalizeTokenDeposit{Sender: "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", Data: []byte("token_deposit_data"), Amount: sdk.NewInt64Coin("init", 10), Height: 5},
				},
				{
					&opchildtypes.MsgRelayOracleData{
						Sender: "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5",
						OracleData: opchildtypes.OracleData{
							BridgeId:        1,
							OraclePriceHash: []byte("price_hash"),
							Prices:          []opchildtypes.OraclePriceData{},
							L1BlockHeight:   100,
							L1BlockTime:     1000000000,
							Proof:           []byte{},
							ProofHeight:     ibcclienttypes.NewHeight(1, 101),
						},
					},
				},
			},
			err: false,
		},
		{
			name: "multiple relay oracle data in same tx",
			txs: [][]sdk.Msg{
				{
					&opchildtypes.MsgRelayOracleData{
						Sender: "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5",
						OracleData: opchildtypes.OracleData{
							BridgeId:        1,
							OraclePriceHash: []byte("price_hash_1"),
							Prices: []opchildtypes.OraclePriceData{
								{CurrencyPair: "BTC/USD", Price: "50000.00", Decimals: 8, Nonce: 1},
							},
							L1BlockHeight: 100,
							L1BlockTime:   1000000000,
							Proof:         []byte("proof_1"),
							ProofHeight:   ibcclienttypes.NewHeight(1, 101),
						},
					},
					&opchildtypes.MsgRelayOracleData{
						Sender: "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5",
						OracleData: opchildtypes.OracleData{
							BridgeId:        2,
							OraclePriceHash: []byte("price_hash_2"),
							Prices: []opchildtypes.OraclePriceData{
								{CurrencyPair: "ETH/USD", Price: "3000.00", Decimals: 8, Nonce: 1},
							},
							L1BlockHeight: 100,
							L1BlockTime:   1000000000,
							Proof:         []byte("proof_2"),
							ProofHeight:   ibcclienttypes.NewHeight(1, 101),
						},
					},
				},
			},
			expectedTxs: [][]sdk.Msg{
				{
					&opchildtypes.MsgRelayOracleData{
						Sender: "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5",
						OracleData: opchildtypes.OracleData{
							BridgeId:        1,
							OraclePriceHash: []byte("price_hash_1"),
							Prices:          []opchildtypes.OraclePriceData{},
							L1BlockHeight:   100,
							L1BlockTime:     1000000000,
							Proof:           []byte{},
							ProofHeight:     ibcclienttypes.NewHeight(1, 101),
						},
					},
					&opchildtypes.MsgRelayOracleData{
						Sender: "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5",
						OracleData: opchildtypes.OracleData{
							BridgeId:        2,
							OraclePriceHash: []byte("price_hash_2"),
							Prices:          []opchildtypes.OraclePriceData{},
							L1BlockHeight:   100,
							L1BlockTime:     1000000000,
							Proof:           []byte{},
							ProofHeight:     ibcclienttypes.NewHeight(1, 101),
						},
					},
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
			name: "already empty prices and proof",
			txs: [][]sdk.Msg{
				{
					&opchildtypes.MsgRelayOracleData{
						Sender: "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5",
						OracleData: opchildtypes.OracleData{
							BridgeId:        1,
							OraclePriceHash: []byte("price_hash"),
							Prices:          []opchildtypes.OraclePriceData{},
							L1BlockHeight:   100,
							L1BlockTime:     1000000000,
							Proof:           []byte{},
							ProofHeight:     ibcclienttypes.NewHeight(1, 101),
						},
					},
				},
			},
			expectedTxs: [][]sdk.Msg{
				{
					&opchildtypes.MsgRelayOracleData{
						Sender: "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5",
						OracleData: opchildtypes.OracleData{
							BridgeId:        1,
							OraclePriceHash: []byte("price_hash"),
							Prices:          []opchildtypes.OraclePriceData{},
							L1BlockHeight:   100,
							L1BlockTime:     1000000000,
							Proof:           []byte{},
							ProofHeight:     ibcclienttypes.NewHeight(1, 101),
						},
					},
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

			changedBlock, err := batchSubmitter.emptyRelayOracleData(&pbb)
			if tc.err {
				require.Error(t, err)
				return
			}
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

func TestEmptyUpdateClientData(t *testing.T) {
	baseDB, err := db.NewMemDB()
	require.NoError(t, err)

	batchDB := baseDB.WithPrefix([]byte("test_batch"))
	appCodec, txConfig, err := childprovider.GetCodec("init")
	require.NoError(t, err)
	batchNode := node.NewTestNode(nodetypes.NodeConfig{}, batchDB, appCodec, txConfig, nil, nil)

	batchSubmitter := BatchSubmitter{node: batchNode}

	createUpdateClientMsg := func(t *testing.T, clientID string, header *ibctmlightclients.Header, sender string) *ibcclienttypes.MsgUpdateClient {
		msg, err := ibcclienttypes.NewMsgUpdateClient(clientID, header, sender)
		require.NoError(t, err)
		return msg
	}

	l1ChainId := "test-host-chain-id"

	cases := []struct {
		name        string
		blocks      []*cmttypes.Block
		txs         [][]sdk.Msg
		expectedTxs [][]sdk.Msg
		err         bool
	}{
		{
			name: "same signatures",
			blocks: []*cmttypes.Block{
				{
					Header: cmttypes.Header{
						ChainID: l1ChainId,
						Height:  6,
					},
					LastCommit: &cmttypes.Commit{
						Signatures: []cmttypes.CommitSig{
							{
								BlockIDFlag:      cmttypes.BlockIDFlagCommit,
								ValidatorAddress: []byte("validator_address"),
								Timestamp:        time.Unix(0, 10000).UTC(),
								Signature:        []byte("signature"),
							},
							{
								BlockIDFlag:      cmttypes.BlockIDFlagCommit,
								ValidatorAddress: []byte("validator_address2"),
								Timestamp:        time.Unix(0, 10001).UTC(),
								Signature:        []byte("signature2"),
							},
							{
								BlockIDFlag:      cmttypes.BlockIDFlagCommit,
								ValidatorAddress: []byte("validator_address3"),
								Timestamp:        time.Unix(0, 10002).UTC(),
								Signature:        []byte("signature3"),
							},
							{
								BlockIDFlag:      cmttypes.BlockIDFlagCommit,
								ValidatorAddress: []byte("validator_address4"),
								Timestamp:        time.Unix(0, 10003).UTC(),
								Signature:        []byte("signature4"),
							},
						},
					},
				},
			},
			txs: [][]sdk.Msg{
				{
					&opchildtypes.MsgFinalizeTokenDeposit{Sender: "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", Data: []byte("token_deposit_data"), Amount: sdk.NewInt64Coin("init", 10), Height: 5},
				},
				{
					createUpdateClientMsg(t, "clientid", &ibctmlightclients.Header{
						SignedHeader: &cmtproto.SignedHeader{
							Header: &cmtproto.Header{
								ChainID: l1ChainId,
								Height:  5,
							},
							Commit: &cmtproto.Commit{
								Signatures: []cmtproto.CommitSig{
									{
										ValidatorAddress: []byte("validator_address"),
										Timestamp:        time.Unix(0, 10000).UTC(),
										Signature:        []byte("signature"),
										BlockIdFlag:      2,
									},
									{
										ValidatorAddress: []byte("validator_address4"),
										Timestamp:        time.Unix(0, 10003).UTC(),
										Signature:        []byte("signature4"),
										BlockIdFlag:      2,
									},
									{
										ValidatorAddress: []byte("validator_address2"),
										Timestamp:        time.Unix(0, 10001).UTC(),
										Signature:        []byte("signature2"),
										BlockIdFlag:      2,
									},
									{
										ValidatorAddress: []byte("validator_address3"),
										Timestamp:        time.Unix(0, 10002).UTC(),
										Signature:        []byte("signature3"),
										BlockIdFlag:      2,
									},
								},
							},
						},
						ValidatorSet: &cmtproto.ValidatorSet{
							Validators: []*cmtproto.Validator{
								{
									Address:     []byte("validator_address"),
									VotingPower: 100,
								},
								{
									Address:     []byte("validator_address2"),
									VotingPower: 200,
								},
								{
									Address:     []byte("validator_address3"),
									VotingPower: 300,
								},
								{
									Address:     []byte("validator_address4"),
									VotingPower: 400,
								},
							},
						},
						TrustedHeight: ibcclienttypes.Height{
							RevisionNumber: 0,
							RevisionHeight: 10,
						},
						TrustedValidators: &cmtproto.ValidatorSet{
							Validators: []*cmtproto.Validator{
								{
									Address:     []byte("validator_address"),
									VotingPower: 100,
								},
								{
									Address:     []byte("validator_address2"),
									VotingPower: 200,
								},
								{
									Address:     []byte("validator_address3"),
									VotingPower: 300,
								},
								{
									Address:     []byte("validator_address4"),
									VotingPower: 400,
								},
							},
						},
					}, "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5"),
				},
			},
			expectedTxs: [][]sdk.Msg{
				{
					&opchildtypes.MsgFinalizeTokenDeposit{Sender: "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", Data: []byte("token_deposit_data"), Amount: sdk.NewInt64Coin("init", 10), Height: 5},
				},
				{
					createUpdateClientMsg(t, "clientid", &ibctmlightclients.Header{
						SignedHeader: &cmtproto.SignedHeader{
							Header: &cmtproto.Header{
								ChainID: l1ChainId,
								Height:  5,
							},
							Commit: &cmtproto.Commit{
								Signatures: []cmtproto.CommitSig{
									{
										ValidatorAddress: []byte{},
										Timestamp:        time.Time{},
										Signature:        []byte{0x00, 0x00},
										BlockIdFlag:      0,
									},
									{
										ValidatorAddress: []byte{},
										Timestamp:        time.Time{},
										Signature:        []byte{0x03, 0x00},
										BlockIdFlag:      0,
									},
									{
										ValidatorAddress: []byte{},
										Timestamp:        time.Time{},
										Signature:        []byte{0x01, 0x00},
										BlockIdFlag:      0,
									},
									{
										ValidatorAddress: []byte{},
										Timestamp:        time.Time{},
										Signature:        []byte{0x02, 0x00},
										BlockIdFlag:      0,
									},
								},
							},
						},
						ValidatorSet: &cmtproto.ValidatorSet{},
						TrustedHeight: ibcclienttypes.Height{
							RevisionNumber: 0,
							RevisionHeight: 10,
						},
						TrustedValidators: &cmtproto.ValidatorSet{},
					}, "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5"),
				},
			},
			err: false,
		},
		{
			name: "different signatures",
			blocks: []*cmttypes.Block{
				{
					Header: cmttypes.Header{
						ChainID: l1ChainId,
						Height:  6,
					},
					LastCommit: &cmttypes.Commit{
						Signatures: []cmttypes.CommitSig{
							{
								BlockIDFlag:      cmttypes.BlockIDFlagCommit,
								ValidatorAddress: []byte("validator_address"),
								Timestamp:        time.Unix(0, 10000).UTC(),
								Signature:        []byte("signature"),
							},
							{
								BlockIDFlag:      cmttypes.BlockIDFlagCommit,
								ValidatorAddress: []byte("validator_address2"),
								Timestamp:        time.Unix(0, 10001).UTC(),
								Signature:        []byte("signature2"),
							},
							{
								BlockIDFlag:      cmttypes.BlockIDFlagCommit,
								ValidatorAddress: []byte("validator_address3"),
								Timestamp:        time.Unix(0, 10002).UTC(),
								Signature:        []byte("signature3"),
							},
							{
								BlockIDFlag:      cmttypes.BlockIDFlagCommit,
								ValidatorAddress: []byte("validator_address4"),
								Timestamp:        time.Unix(0, 10003).UTC(),
								Signature:        []byte("signature4"),
							},
						},
					},
				},
			},
			txs: [][]sdk.Msg{
				{
					&opchildtypes.MsgFinalizeTokenDeposit{Sender: "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", Data: []byte("token_deposit_data"), Amount: sdk.NewInt64Coin("init", 10), Height: 5},
				},
				{
					createUpdateClientMsg(t, "clientid", &ibctmlightclients.Header{
						SignedHeader: &cmtproto.SignedHeader{
							Header: &cmtproto.Header{
								ChainID: l1ChainId,
								Height:  5,
							},
							Commit: &cmtproto.Commit{
								Signatures: []cmtproto.CommitSig{
									{
										ValidatorAddress: []byte("validator_address"),
										Timestamp:        time.Unix(0, 10001).UTC(),
										Signature:        []byte("signature"),
										BlockIdFlag:      2,
									},
									{
										ValidatorAddress: []byte("validator_address2"),
										Timestamp:        time.Unix(0, 10001).UTC(),
										Signature:        []byte("signature22"),
										BlockIdFlag:      2,
									},
									{
										ValidatorAddress: []byte{},
										Timestamp:        time.Time{},
										Signature:        []byte{},
										BlockIdFlag:      0,
									},
									{
										ValidatorAddress: []byte("validator_address4"),
										Timestamp:        time.Unix(0, 10003).UTC(),
										Signature:        []byte("signature4"),
										BlockIdFlag:      1,
									},
								},
							},
						},
						ValidatorSet: &cmtproto.ValidatorSet{
							Validators: []*cmtproto.Validator{
								{
									Address:     []byte("validator_address"),
									VotingPower: 100,
								},
								{
									Address:     []byte("validator_address2"),
									VotingPower: 200,
								},
								{
									Address:     []byte("validator_address3"),
									VotingPower: 300,
								},
								{
									Address:     []byte("validator_address4"),
									VotingPower: 400,
								},
							},
						},
						TrustedHeight: ibcclienttypes.Height{
							RevisionNumber: 0,
							RevisionHeight: 10,
						},
						TrustedValidators: &cmtproto.ValidatorSet{
							Validators: []*cmtproto.Validator{
								{
									Address:     []byte("validator_address"),
									VotingPower: 100,
								},
								{
									Address:     []byte("validator_address2"),
									VotingPower: 200,
								},
								{
									Address:     []byte("validator_address3"),
									VotingPower: 300,
								},
								{
									Address:     []byte("validator_address4"),
									VotingPower: 400,
								},
							},
						},
					}, "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5"),
				},
			},
			expectedTxs: [][]sdk.Msg{
				{
					&opchildtypes.MsgFinalizeTokenDeposit{Sender: "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", Data: []byte("token_deposit_data"), Amount: sdk.NewInt64Coin("init", 10), Height: 5},
				},
				{
					createUpdateClientMsg(t, "clientid", &ibctmlightclients.Header{
						SignedHeader: &cmtproto.SignedHeader{
							Header: &cmtproto.Header{
								ChainID: l1ChainId,
								Height:  5,
							},
							Commit: &cmtproto.Commit{
								Signatures: []cmtproto.CommitSig{
									{
										ValidatorAddress: []byte("validator_address"),
										Timestamp:        time.Unix(0, 10001).UTC(),
										Signature:        []byte("signature"),
										BlockIdFlag:      2,
									},
									{
										ValidatorAddress: []byte("validator_address2"),
										Timestamp:        time.Unix(0, 10001).UTC(),
										Signature:        []byte("signature22"),
										BlockIdFlag:      2,
									},
									{
										ValidatorAddress: []byte{},
										Timestamp:        time.Time{},
										Signature:        []byte{},
										BlockIdFlag:      0,
									},
									{
										ValidatorAddress: []byte("validator_address4"),
										Timestamp:        time.Unix(0, 10003).UTC(),
										Signature:        []byte("signature4"),
										BlockIdFlag:      1,
									},
								},
							},
						},
						ValidatorSet: &cmtproto.ValidatorSet{},
						TrustedHeight: ibcclienttypes.Height{
							RevisionNumber: 0,
							RevisionHeight: 10,
						},
						TrustedValidators: &cmtproto.ValidatorSet{},
					}, "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5"),
				},
			},
			err: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockHost := NewMockHost(nil, l1ChainId)
			batchSubmitter.host = mockHost

			for _, block := range tc.blocks {
				mockHost.SetBlock(block.Height, block)
			}

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

			ctx := types.NewContext(context.TODO(), zap.NewNop(), "").WithPollingInterval(time.Millisecond)
			changedBlock, err := batchSubmitter.emptyUpdateClientData(ctx, &pbb)
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
