package types_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	childprovider "github.com/initia-labs/opinit-bots/provider/child"

	"github.com/stretchr/testify/require"
)

func Test_MarshalUnMarshal_ProcessedMsgs(t *testing.T) {
	appCodec, _, err := childprovider.GetCodec("init")
	require.NoError(t, err)

	msgs := btypes.ProcessedMsgs{
		Msgs:      []sdk.Msg{},
		Timestamp: time.Now().UnixMicro(),
		Save:      true,
	}

	bz, err := msgs.MarshalInterfaceJSON(appCodec)
	require.NoError(t, err)

	var msgs2 btypes.ProcessedMsgs
	err = msgs2.UnmarshalInterfaceJSON(appCodec, bz)
	require.NoError(t, err)

	require.Equal(t, msgs, msgs2)
}
