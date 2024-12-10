package host

import (
	"testing"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"
	"github.com/initia-labs/opinit-bots/db"
	"github.com/initia-labs/opinit-bots/node"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	childprovider "github.com/initia-labs/opinit-bots/provider/child"
	hostprovider "github.com/initia-labs/opinit-bots/provider/host"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestOracleTxHandler(t *testing.T) {
	db, err := db.NewMemDB()
	require.NoError(t, err)
	hostNode := node.NewTestNode(nodetypes.NodeConfig{}, db.WithPrefix([]byte("test_host")), nil, nil, nil, nil)
	childCodec, _, err := childprovider.GetCodec("init")

	h := Host{
		child: NewMockChild(db.WithPrefix([]byte("test_child")), childCodec, "init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", 1),
	}

	cases := []struct {
		name          string
		oracleEnabled bool
		blockHeight   int64
		extCommitBz   []byte
		expected      func() (sender string, msg sdk.Msg, err error)
		err           bool
	}{
		{
			name:          "oracle enabled",
			oracleEnabled: true,
			blockHeight:   3,
			extCommitBz:   []byte("oracle_tx"),
			expected: func() (sender string, msg sdk.Msg, err error) {
				msg, err = childprovider.CreateAuthzMsg("init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5", opchildtypes.NewMsgUpdateOracle("init1z3689ct7pc72yr5an97nsj89dnlefydxwdhcv0", 3, []byte("oracle_tx")))
				sender = "init1hrasklz3tr6s9rls4r8fjuf0k4zuha6w9rude5"
				return sender, msg, err
			},
			err: false,
		},
		{
			name:          "oracle disabled",
			oracleEnabled: false,
			blockHeight:   3,
			extCommitBz:   []byte("valid_oracle_tx"),
			expected:      nil,
			err:           false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h.BaseHost = hostprovider.NewTestBaseHost(0, hostNode, ophosttypes.QueryBridgeResponse{
				BridgeId: 1,
				BridgeConfig: ophosttypes.BridgeConfig{
					OracleEnabled: tc.oracleEnabled,
				},
			}, nodetypes.NodeConfig{}, nil)

			msg, sender, err := h.oracleTxHandler(tc.blockHeight, tc.extCommitBz)
			if !tc.err {
				require.NoError(t, err)
				if tc.expected != nil {
					expectedSender, expectedMsg, err := tc.expected()
					require.NoError(t, err)
					require.Equal(t, expectedSender, sender)
					require.Equal(t, expectedMsg, msg)
				} else {
					require.Nil(t, msg)
				}
			} else {
				require.Error(t, err)
			}
			h.EmptyProcessedMsgs()
		})
	}
	require.NoError(t, err)
}
