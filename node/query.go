package node

import (
	"context"
	"fmt"
	"reflect"
	"strconv"

	sdkerrors "cosmossdk.io/errors"
	abci "github.com/cometbft/cometbft/abci/types"
	client2 "github.com/cometbft/cometbft/rpc/client"
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	grpctypes "github.com/cosmos/cosmos-sdk/types/grpc"
	"github.com/cosmos/cosmos-sdk/types/tx"
	gogogrpc "github.com/cosmos/gogoproto/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/encoding/proto"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var _ gogogrpc.ClientConn = &Node{}

var protoCodec = encoding.GetCodec(proto.Name)

// Invoke implements the grpc ClientConn.Invoke method
func (n *Node) Invoke(ctx context.Context, method string, req, reply interface{}, opts ...grpc.CallOption) (err error) {
	// Two things can happen here:
	// 1. either we're broadcasting a Tx, in which call we call Tendermint's broadcast endpoint directly,
	// 2. or we are querying for state, in which case we call ABCI's Querier.

	// In both cases, we don't allow empty request req (it will panic unexpectedly).
	if reflect.ValueOf(req).IsNil() {
		return sdkerrors.Wrap(legacyerrors.ErrInvalidRequest, "request cannot be nil")
	}

	// Case 1. Broadcasting a Tx.
	if reqProto, ok := req.(*tx.BroadcastTxRequest); ok {
		if !ok {
			return sdkerrors.Wrapf(legacyerrors.ErrInvalidRequest, "expected %T, got %T", (*tx.BroadcastTxRequest)(nil), req)
		}
		resProto, ok := reply.(*tx.BroadcastTxResponse)
		if !ok {
			return sdkerrors.Wrapf(legacyerrors.ErrInvalidRequest, "expected %T, got %T", (*tx.BroadcastTxResponse)(nil), req)
		}

		broadcastRes, err := n.TxServiceBroadcast(ctx, reqProto)
		if err != nil {
			return err
		}
		*resProto = *broadcastRes
		return err
	}

	// Case 2. Querying state.
	inMd, _ := metadata.FromOutgoingContext(ctx)
	abciRes, outMd, err := n.RunGRPCQuery(ctx, method, req, inMd)
	if err != nil {
		return err
	}

	if err = protoCodec.Unmarshal(abciRes.Value, reply); err != nil {
		return err
	}

	for _, callOpt := range opts {
		header, ok := callOpt.(grpc.HeaderCallOption)
		if !ok {
			continue
		}

		*header.HeaderAddr = outMd
	}

	if n.cdc.InterfaceRegistry() != nil {
		return types.UnpackInterfaces(reply, n.cdc)
	}

	return nil
}

// TxServiceBroadcast is a helper function to broadcast a Tx with the correct gRPC types
// from the tx service. Calls `clientCtx.BroadcastTx` under the hood.
func (n *Node) TxServiceBroadcast(ctx context.Context, req *tx.BroadcastTxRequest) (*tx.BroadcastTxResponse, error) {
	if req == nil || req.TxBytes == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid empty tx")
	}

	// TODO: use sync & wait tx until it is included in a block
	res, err := n.BroadcastTxCommit(ctx, req.TxBytes)
	if err != nil {
		return nil, err
	}

	return &tx.BroadcastTxResponse{
		TxResponse: &sdk.TxResponse{
			Height:    res.Height,
			TxHash:    res.Hash.String(),
			Codespace: res.TxResult.Codespace,
			Code:      res.TxResult.Code,
			Data:      string(res.TxResult.Data),
		},
	}, nil
}

// NewStream implements the grpc ClientConn.NewStream method
func (n *Node) NewStream(context.Context, *grpc.StreamDesc, string, ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, fmt.Errorf("streaming rpc not supported")
}

// RunGRPCQuery runs a gRPC query from the clientCtx, given all necessary
// arguments for the gRPC method, and returns the ABCI response. It is used
// to factorize code between client (Invoke) and server (RegisterGRPCServer)
// gRPC handlers.
func (n *Node) RunGRPCQuery(ctx context.Context, method string, req interface{}, md metadata.MD) (abci.ResponseQuery, metadata.MD, error) {
	reqBz, err := protoCodec.Marshal(req)
	if err != nil {
		return abci.ResponseQuery{}, nil, err
	}

	// parse height header
	if heights := md.Get(grpctypes.GRPCBlockHeightHeader); len(heights) > 0 {
		height, err := strconv.ParseInt(heights[0], 10, 64)
		if err != nil {
			return abci.ResponseQuery{}, nil, err
		}
		if height < 0 {
			return abci.ResponseQuery{}, nil, sdkerrors.Wrapf(
				legacyerrors.ErrInvalidRequest,
				"client.Context.Invoke: height (%d) from %q must be >= 0", height, grpctypes.GRPCBlockHeightHeader)
		}

	}

	height, err := GetHeightFromMetadata(md)
	if err != nil {
		return abci.ResponseQuery{}, nil, err
	}

	abciReq := abci.RequestQuery{
		Path:   method,
		Data:   reqBz,
		Height: height,
		Prove:  false,
	}

	abciRes, err := n.QueryABCI(ctx, abciReq)
	if err != nil {
		return abci.ResponseQuery{}, nil, err
	}

	// Create header metadata. For now the headers contain:
	// - block height
	// We then parse all the call options, if the call option is a
	// HeaderCallOption, then we manually set the value of that header to the
	// metadata.
	md = metadata.Pairs(grpctypes.GRPCBlockHeightHeader, strconv.FormatInt(abciRes.Height, 10))

	return abciRes, md, nil
}

// QueryABCI performs an ABCI query and returns the appropriate response and error sdk error code.
func (n *Node) QueryABCI(ctx context.Context, req abci.RequestQuery) (abci.ResponseQuery, error) {
	opts := client2.ABCIQueryOptions{
		Height: req.Height,
		Prove:  req.Prove,
	}

	result, err := n.ABCIQueryWithOptions(ctx, req.Path, req.Data, opts)
	if err != nil {
		return abci.ResponseQuery{}, err
	}

	if !result.Response.IsOK() {
		return abci.ResponseQuery{}, sdkerrors.ABCIError(result.Response.Codespace, result.Response.Code, result.Response.Log)
	}

	return result.Response, nil
}

func GetHeightFromMetadata(md metadata.MD) (int64, error) {
	height := md.Get(grpctypes.GRPCBlockHeightHeader)
	if len(height) == 1 {
		return strconv.ParseInt(height[0], 10, 64)
	}
	return 0, nil
}
