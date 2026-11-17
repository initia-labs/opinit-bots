package batchsubmitter

import (
	"context"
	"testing"
	"time"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ibcclienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ibctmlightclients "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	"github.com/initia-labs/opinit-bots/db"
	"github.com/initia-labs/opinit-bots/node"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	childprovider "github.com/initia-labs/opinit-bots/provider/child"
	"github.com/initia-labs/opinit-bots/txutils"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// countingHost counts block queries; it serves only the blocks it was given, like a pruned or state-synced host node.
type countingHost struct {
	*mockHost
	calls int
}

func (c *countingHost) QueryBlock(ctx context.Context, height int64) (*coretypes.ResultBlock, error) {
	c.calls++
	return c.mockHost.QueryBlock(ctx, height)
}

func updateClientBlock(t *testing.T, hostChainID string, headerHeight int64) (*BatchSubmitter, *cmtproto.Block, []byte) {
	baseDB, err := db.NewMemDB()
	require.NoError(t, err)
	appCodec, txConfig, err := childprovider.GetCodec("init")
	require.NoError(t, err)
	batchNode := node.NewTestNode(nodetypes.NodeConfig{}, baseDB.WithPrefix([]byte("test_batch")), appCodec, txConfig, nil, nil)
	bs := &BatchSubmitter{node: batchNode}

	header := &ibctmlightclients.Header{
		SignedHeader: &cmtproto.SignedHeader{
			Header: &cmtproto.Header{ChainID: hostChainID, Height: headerHeight},
			Commit: &cmtproto.Commit{Signatures: []cmtproto.CommitSig{{
				ValidatorAddress: []byte("validator_address"), Timestamp: time.Unix(0, 10000).UTC(), Signature: []byte("signature"), BlockIdFlag: 2,
			}}},
		},
		ValidatorSet:      &cmtproto.ValidatorSet{Validators: []*cmtproto.Validator{{Address: []byte("validator_address"), VotingPower: 100}}},
		TrustedHeight:     ibcclienttypes.Height{RevisionNumber: 0, RevisionHeight: 1},
		TrustedValidators: &cmtproto.ValidatorSet{Validators: []*cmtproto.Validator{{Address: []byte("validator_address"), VotingPower: 100}}},
	}
	msg, err := ibcclienttypes.NewMsgUpdateClient("07-tendermint-0", header, "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5")
	require.NoError(t, err)
	txb, err := tx.Factory{}.WithChainID("test_chain").WithTxConfig(txConfig).BuildUnsignedTx([]sdk.Msg{msg}...)
	require.NoError(t, err)
	txBytes, err := txutils.EncodeTx(txConfig, txb.GetTx())
	require.NoError(t, err)
	return bs, &cmtproto.Block{Data: cmtproto.Data{Txs: [][]byte{txBytes}}}, txBytes
}

// host node that does not hold the old block (state-synced after it) must not stop the batch submitter
func TestUpdateClientKeptWhenHostBlockUnavailable(t *testing.T) {
	hostChainID := "interwoven-1"
	bs, pbb, original := updateClientBlock(t, hostChainID, 23641927)
	host := &countingHost{mockHost: NewMockHost(nil, hostChainID)} // no blocks at all
	bs.host = host

	ctx := types.NewContext(context.TODO(), zap.NewNop(), "").WithPollingInterval(time.Millisecond)
	out, err := bs.emptyUpdateClientData(ctx, pbb)
	require.NoError(t, err, "a host block that cannot be served must not fail the L2 block")
	require.Equal(t, original, out.Data.Txs[0], "the message must be left exactly as it was")
	require.Equal(t, types.MaxRetryCount, host.calls)
}

// first usable answer ends the query loop.
func TestUpdateClientQueriesHostBlockOnce(t *testing.T) {
	hostChainID := "interwoven-1"
	bs, pbb, original := updateClientBlock(t, hostChainID, 5)
	host := &countingHost{mockHost: NewMockHost(nil, hostChainID)}
	host.SetBlock(6, &cmttypes.Block{
		Header: cmttypes.Header{ChainID: hostChainID, Height: 6},
		LastCommit: &cmttypes.Commit{Signatures: []cmttypes.CommitSig{{
			BlockIDFlag: cmttypes.BlockIDFlagCommit, ValidatorAddress: []byte("validator_address"), Timestamp: time.Unix(0, 10000).UTC(), Signature: []byte("signature"),
		}}},
	})
	bs.host = host

	ctx := types.NewContext(context.TODO(), zap.NewNop(), "").WithPollingInterval(time.Millisecond)
	out, err := bs.emptyUpdateClientData(ctx, pbb)
	require.NoError(t, err)
	require.NotEqual(t, original, out.Data.Txs[0], "with the host block available the message is shortened as before")
	require.Equal(t, 1, host.calls, "one successful query is enough")
}
