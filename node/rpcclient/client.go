package rpcclient

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/encoding/proto"
	"google.golang.org/grpc/metadata"

	sdkerrors "cosmossdk.io/errors"
	abci "github.com/cometbft/cometbft/abci/types"
	client2 "github.com/cometbft/cometbft/rpc/client"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	cmttypes "github.com/cometbft/cometbft/types"
	gogogrpc "github.com/cosmos/gogoproto/grpc"
	"github.com/pkg/errors"
	"go.uber.org/zap"

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

	cdc  codec.Codec
	pool *RPCPool
	mu   sync.RWMutex
}

func NewRPCClient(cdc codec.Codec, rpcAddrs []string, logger *zap.Logger) (*RPCClient, error) {
	if len(rpcAddrs) == 0 {
		return nil, errors.New("no RPC addresses provided")
	}

	// Create RPC pool
	pool := NewRPCPool(rpcAddrs, logger)

	// Create HTTP client with the first endpoint
	client, err := clienthttp.New(pool.GetCurrentEndpoint(), "/websocket")
	if err != nil {
		return nil, err
	}

	return &RPCClient{
		HTTP: client,
		cdc:  cdc,
		pool: pool,
	}, nil
}

func NewRPCClientWithClient(cdc codec.Codec, client *clienthttp.HTTP, endpoints []string, logger *zap.Logger) (*RPCClient, error) {
	if len(endpoints) == 0 {
		return nil, errors.New("no RPC endpoints provided")
	}

	// Create RPC pool
	pool := NewRPCPool(endpoints, logger)

	return &RPCClient{
		HTTP: client,
		cdc:  cdc,
		pool: pool,
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

	var result *coretypes.ResultABCIQuery
	var err error

	execErr := q.ExecuteWithFallback(ctx, func(ctx context.Context) error {
		result, err = q.ABCIQueryWithOptions(ctx, req.Path, req.Data, opts)
		return err
	})

	if execErr != nil {
		return abci.ResponseQuery{}, execErr
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

	var result []byte
	var err error

	execErr := q.ExecuteWithFallback(ctx, func(ctx context.Context) error {
		result, err = q.RawCommit(ctx, &height)
		return err
	})

	if execErr != nil {
		return nil, execErr
	}

	return result, nil
}

// QueryBlockBulk queries blocks in bulk with fallback and retry logic.
func (q *RPCClient) QueryBlockBulk(ctx context.Context, start int64, end int64) ([][]byte, error) {
	ctx, cancel := GetQueryContext(ctx, 0)
	defer cancel()

	var result [][]byte
	var err error

	execErr := q.ExecuteWithFallback(ctx, func(ctx context.Context) error {
		result, err = q.BlockBulk(ctx, &start, &end)
		return err
	})

	if execErr != nil {
		return nil, execErr
	}

	return result, nil
}

// ExecuteWithFallback executes the given function with fallback to other endpoints if it fails
func (q *RPCClient) ExecuteWithFallback(ctx context.Context, fn func(context.Context) error) error {
	return q.pool.ExecuteWithFallback(ctx, func(ctx context.Context) error {
		// Update HTTP client to current endpoint before executing
		if err := q.updateHTTPClient(); err != nil {
			return err
		}
		return fn(ctx)
	})
}

// updateHTTPClient updates the HTTP client to use the current endpoint from the pool
func (q *RPCClient) updateHTTPClient() error {
	// If this is a mock client (created with NewWithCaller), don't replace it
	// Mock clients have empty remote and nil rpc field
	if q.HTTP.Remote() == "" {
		return nil
	}

	currentEndpoint := q.pool.GetCurrentEndpoint()
	if q.HTTP.Remote() != currentEndpoint {
		// Create new HTTP client with current endpoint
		client, err := clienthttp.New(currentEndpoint, "/websocket")
		if err != nil {
			return err
		}
		q.mu.Lock()
		q.HTTP = client
		q.mu.Unlock()
	}
	return nil
}

// Status returns the status of the node with fallback and retry logic
func (q *RPCClient) Status(ctx context.Context) (*coretypes.ResultStatus, error) {
	var result *coretypes.ResultStatus
	var err error

	execErr := q.ExecuteWithFallback(ctx, func(ctx context.Context) error {
		result, err = q.HTTP.Status(ctx)
		return err
	})

	if execErr != nil {
		return nil, execErr
	}

	return result, nil
}

// Block returns the block at the given height with fallback and retry logic
func (q *RPCClient) Block(ctx context.Context, height *int64) (*coretypes.ResultBlock, error) {
	var result *coretypes.ResultBlock
	var err error

	execErr := q.ExecuteWithFallback(ctx, func(ctx context.Context) error {
		result, err = q.HTTP.Block(ctx, height)
		return err
	})

	if execErr != nil {
		return nil, execErr
	}

	return result, nil
}

// BlockResults returns the block results at the given height with fallback and retry logic
func (q *RPCClient) BlockResults(ctx context.Context, height *int64) (*coretypes.ResultBlockResults, error) {
	var result *coretypes.ResultBlockResults
	var err error

	execErr := q.ExecuteWithFallback(ctx, func(ctx context.Context) error {
		result, err = q.HTTP.BlockResults(ctx, height)
		return err
	})

	if execErr != nil {
		return nil, execErr
	}

	return result, nil
}

func (q *RPCClient) QueryTx(ctx context.Context, txHash []byte) (*coretypes.ResultTx, error) {
	ctx, cancel := GetQueryContext(ctx, 0)
	defer cancel()

	var result *coretypes.ResultTx
	var err error

	execErr := q.ExecuteWithFallback(ctx, func(ctx context.Context) error {
		result, err = q.Tx(ctx, txHash, false)
		return err
	})

	if execErr != nil {
		return nil, execErr
	}

	return result, nil
}

// Tx returns the transaction with the given hash with fallback and retry logic
func (q *RPCClient) Tx(ctx context.Context, hash []byte, prove bool) (*coretypes.ResultTx, error) {
	var result *coretypes.ResultTx
	var err error

	execErr := q.ExecuteWithFallback(ctx, func(ctx context.Context) error {
		result, err = q.HTTP.Tx(ctx, hash, prove)
		return err
	})

	if execErr != nil {
		return nil, execErr
	}

	return result, nil
}

// BroadcastTxSync broadcasts a transaction synchronously with fallback and retry logic
func (q *RPCClient) BroadcastTxSync(ctx context.Context, tx cmttypes.Tx) (*coretypes.ResultBroadcastTx, error) {
	var result *coretypes.ResultBroadcastTx
	var err error

	execErr := q.ExecuteWithFallback(ctx, func(ctx context.Context) error {
		result, err = q.HTTP.BroadcastTxSync(ctx, tx)
		return err
	})

	if execErr != nil {
		return nil, execErr
	}

	return result, nil
}

// BroadcastTxAsync broadcasts a transaction asynchronously with fallback and retry logic
func (q *RPCClient) BroadcastTxAsync(ctx context.Context, tx cmttypes.Tx) (*coretypes.ResultBroadcastTx, error) {
	var result *coretypes.ResultBroadcastTx
	var err error

	execErr := q.ExecuteWithFallback(ctx, func(ctx context.Context) error {
		result, err = q.HTTP.BroadcastTxAsync(ctx, tx)
		return err
	})

	if execErr != nil {
		return nil, execErr
	}

	return result, nil
}
