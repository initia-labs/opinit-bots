package rpcclient

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// Since gRPC-Go v1.66 the "proto" codec is registered only as a CodecV2. A nil
// codec here makes every RPCClient query panic, so guard the lookup and wire format.
func TestProtoCodec(t *testing.T) {
	require.NotNil(t, protoCodec)

	req := &banktypes.QuerySupplyOfRequest{Denom: "uinit"}
	bz, err := marshalProto(req)
	require.NoError(t, err)

	gogoBz, err := req.Marshal()
	require.NoError(t, err)
	require.Equal(t, gogoBz, bz)

	res := &banktypes.QuerySupplyOfResponse{Amount: sdk.NewCoin("uinit", math.NewInt(1_000_000_000_000_000))}
	resBz, err := res.Marshal()
	require.NoError(t, err)

	var decoded banktypes.QuerySupplyOfResponse
	require.NoError(t, unmarshalProto(resBz, &decoded))
	require.True(t, res.Amount.Equal(decoded.Amount))
}
