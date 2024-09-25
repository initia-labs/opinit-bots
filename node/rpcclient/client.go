package rpcclient

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/encoding/proto"
	"google.golang.org/grpc/metadata"

	sdkerrors "cosmossdk.io/errors"
	abci "github.com/cometbft/cometbft/abci/types"
	client2 "github.com/cometbft/cometbft/rpc/client"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	gogogrpc "github.com/cosmos/gogoproto/grpc"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	grpctypes "github.com/cosmos/cosmos-sdk/types/grpc"

	clienthttp "github.com/initia-labs/opinit-bots/client"
)

var _ gogogrpc.ClientConn = &RPCClient{}

var protoCodec = encoding.GetCodec(proto.Name)

// RPCClient defines a gRPC querier client.
type RPCClient struct {
	*clienthttp.HTTP

	cdc codec.Codec
}

func NewRPCClient(cdc codec.Codec, rpcAddr string) (*RPCClient, error) {
	client, err := clienthttp.New(rpcAddr, "/websocket")
	if err != nil {
		return nil, err
	}

	return &RPCClient{
		HTTP: client,
		cdc:  cdc,
	}, nil
}

// Invoke implements the grpc ClientConq.Invoke method
func (q RPCClient) Invoke(ctx context.Context, method string, req, reply interface{}, opts ...grpc.CallOption) (err error) {
	// In both cases, we don't allow empty request req (it will panic unexpectedly).
	if reflect.ValueOf(req).IsNil() {
		return sdkerrors.Wrap(legacyerrors.ErrInvalidRequest, "request cannot be nil")
	}

	inMd, _ := metadata.FromOutgoingContext(ctx)
	abciRes, outMd, err := q.RunGRPCQuery(ctx, method, req, inMd)
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

	if q.cdc.InterfaceRegistry() != nil {
		return types.UnpackInterfaces(reply, q.cdc)
	}

	return nil
}

// NewStream implements the grpc ClientConq.NewStream method
func (q RPCClient) NewStream(context.Context, *grpc.StreamDesc, string, ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, fmt.Errorf("streaming rpc not supported")
}

// RunGRPCQuery runs a gRPC query from the clientCtx, given all necessary
// arguments for the gRPC method, and returns the ABCI response. It is used
// to factorize code between client (Invoke) and server (RegisterGRPCServer)
// gRPC handlers.
func (q RPCClient) RunGRPCQuery(ctx context.Context, method string, req interface{}, md metadata.MD) (abci.ResponseQuery, metadata.MD, error) {
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

	abciRes, err := q.QueryABCI(ctx, abciReq)
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
func (q RPCClient) QueryABCI(ctx context.Context, req abci.RequestQuery) (abci.ResponseQuery, error) {
	opts := client2.ABCIQueryOptions{
		Height: req.Height,
		Prove:  req.Prove,
	}

	result, err := q.ABCIQueryWithOptions(ctx, req.Path, req.Data, opts)
	if err != nil {
		return abci.ResponseQuery{}, err
	}

	if !result.Response.IsOK() {
		return abci.ResponseQuery{}, errors.New(result.Response.Log)
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

// Canceling this context releases resources associated with it, so code should
// call cancel as soon as the operations running in this [Context] complete:
func GetQueryContext(ctx context.Context, height int64) (context.Context, context.CancelFunc) {
	// TODO: configurable timeout
	timeout := 10 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)

	strHeight := strconv.FormatInt(height, 10)
	ctx = metadata.AppendToOutgoingContext(ctx, grpctypes.GRPCBlockHeightHeader, strHeight)
	return ctx, cancel
}

// QueryRawCommit queries the raw commit at a given height.
func (q RPCClient) QueryRawCommit(ctx context.Context, height int64) ([]byte, error) {
	ctx, cancel := GetQueryContext(ctx, height)
	defer cancel()
	return q.RawCommit(ctx, &height)
}

// QueryBlockBulk queries blocks in bulk.
func (q RPCClient) QueryBlockBulk(ctx context.Context, start int64, end int64) ([][]byte, error) {
	ctx, cancel := GetQueryContext(ctx, 0)
	defer cancel()
	return q.BlockBulk(ctx, &start, &end)
}

func (q RPCClient) QueryTx(ctx context.Context, txHash []byte) (*coretypes.ResultTx, error) {
	ctx, cancel := GetQueryContext(ctx, 0)
	defer cancel()
	return q.Tx(ctx, txHash, false)
}
