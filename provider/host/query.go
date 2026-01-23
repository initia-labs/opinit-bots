package host

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	cmtypes "github.com/cometbft/cometbft/types"
	connecttypes "github.com/skip-mev/connect/v2/pkg/types"
	oracletypes "github.com/skip-mev/connect/v2/x/oracle/types"

	query "github.com/cosmos/cosmos-sdk/types/query"

	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"

	"github.com/initia-labs/opinit-bots/node/rpcclient"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/pkg/errors"
)

func (b BaseHost) QueryBridgeConfig(ctx context.Context, bridgeId uint64) (*ophosttypes.QueryBridgeResponse, error) {
	req := &ophosttypes.QueryBridgeRequest{
		BridgeId: bridgeId,
	}
	ctx, cancel := rpcclient.GetQueryContext(ctx, 0)
	defer cancel()

	return b.ophostQueryClient.Bridge(ctx, req)
}

func (b BaseHost) QueryLastFinalizedOutput(ctx context.Context, bridgeId uint64) (*ophosttypes.QueryLastFinalizedOutputResponse, error) {
	req := &ophosttypes.QueryLastFinalizedOutputRequest{
		BridgeId: bridgeId,
	}
	ctx, cancel := rpcclient.GetQueryContext(ctx, 0)
	defer cancel()

	res, err := b.ophostQueryClient.LastFinalizedOutput(ctx, req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, nil
		}
		return nil, err
	}
	return res, nil
}

func (b BaseHost) QueryLastOutput(ctx context.Context, bridgeId uint64) (*ophosttypes.QueryOutputProposalResponse, error) {
	req := &ophosttypes.QueryOutputProposalsRequest{
		BridgeId: bridgeId,
		Pagination: &query.PageRequest{
			Limit:   1,
			Reverse: true,
		},
	}
	ctx, cancel := rpcclient.GetQueryContext(ctx, 0)
	defer cancel()

	res, err := b.ophostQueryClient.OutputProposals(ctx, req)
	if err != nil {
		return nil, err
	}
	if res == nil || res.OutputProposals == nil || len(res.OutputProposals) == 0 {
		return nil, nil
	}
	return &res.OutputProposals[0], nil
}

func (b BaseHost) QueryOutput(ctx context.Context, bridgeId uint64, outputIndex uint64, height int64) (*ophosttypes.QueryOutputProposalResponse, error) {
	req := &ophosttypes.QueryOutputProposalRequest{
		BridgeId:    bridgeId,
		OutputIndex: outputIndex,
	}
	ctx, cancel := rpcclient.GetQueryContext(ctx, height)
	defer cancel()

	return b.ophostQueryClient.OutputProposal(ctx, req)
}

// QueryOutputByL2BlockNumber queries the last output proposal before the given L2 block number
func (b BaseHost) QueryOutputByL2BlockNumber(ctx context.Context, bridgeId uint64, l2BlockHeight int64) (*ophosttypes.QueryOutputProposalResponse, error) {
	start, err := b.QueryOutput(ctx, bridgeId, 1, 0)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, nil
		}
		return nil, err
	}
	end, err := b.QueryLastOutput(ctx, bridgeId)
	if err != nil {
		return nil, err
	} else if end == nil {
		return nil, nil
	}

	l2BlockNumber := types.MustInt64ToUint64(l2BlockHeight)
	for {
		if start.OutputProposal.L2BlockNumber >= l2BlockNumber {
			if start.OutputIndex != 1 {
				return b.QueryOutput(ctx, bridgeId, start.OutputIndex-1, 0)
			}
			return nil, nil
		} else if end.OutputProposal.L2BlockNumber < l2BlockNumber {
			return end, nil
		} else if end.OutputIndex-start.OutputIndex <= 1 {
			return start, nil
		}

		midIndex := (start.OutputIndex + end.OutputIndex) / 2
		output, err := b.QueryOutput(ctx, bridgeId, midIndex, 0)
		if err != nil {
			return nil, err
		}

		if output.OutputProposal.L2BlockNumber <= l2BlockNumber {
			start = output
		} else {
			end = output
		}
	}
}

func (b BaseHost) QueryCreateBridgeHeight(ctx context.Context, bridgeId uint64) (int64, error) {
	ctx, cancel := rpcclient.GetQueryContext(ctx, 0)
	defer cancel()

	query := fmt.Sprintf("%s.%s = %d",
		ophosttypes.EventTypeCreateBridge,
		ophosttypes.AttributeKeyBridgeId,
		bridgeId,
	)
	perPage := 1
	res, err := b.node.GetRPCClient().TxSearch(ctx, query, false, nil, &perPage, "desc")
	if err != nil {
		return 0, err
	}
	if len(res.Txs) == 0 {
		// bridge not found
		return 0, errors.New("bridge not found")
	}
	return res.Txs[0].Height, nil
}

func (b BaseHost) QueryBatchInfos(botCtx types.Context, bridgeId uint64) ([]ophosttypes.BatchInfoWithOutput, error) {
	ctx, cancel := rpcclient.GetQueryContext(botCtx, 0)
	defer cancel()

	ticker := time.NewTicker(botCtx.PollingInterval())
	defer ticker.Stop()

	var batchInfos []ophosttypes.BatchInfoWithOutput
	var nextKey []byte
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}

		req := &ophosttypes.QueryBatchInfosRequest{
			BridgeId: bridgeId,
			Pagination: &query.PageRequest{
				Limit: 100,
				Key:   nextKey,
			},
		}
		res, err := b.ophostQueryClient.BatchInfos(ctx, req)
		if err != nil {
			return nil, err
		}
		batchInfos = append(batchInfos, res.BatchInfos...)
		nextKey = res.Pagination.NextKey
		if nextKey == nil {
			break
		}
	}
	return batchInfos, nil
}

func (b BaseHost) QueryDepositTxHeight(botCtx types.Context, bridgeId uint64, l1Sequence uint64) (int64, error) {
	if l1Sequence == 0 {
		return 0, nil
	}

	ticker := time.NewTicker(botCtx.PollingInterval())
	defer ticker.Stop()

	ctx, cancel := rpcclient.GetQueryContext(botCtx, 0)
	defer cancel()

	query := fmt.Sprintf("%s.%s = %d",
		ophosttypes.EventTypeInitiateTokenDeposit,
		ophosttypes.AttributeKeyL1Sequence,
		l1Sequence,
	)
	per_page := 100
	for page := 1; ; page++ {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-ticker.C:
		}

		res, err := b.node.GetRPCClient().TxSearch(ctx, query, false, &page, &per_page, "asc")
		if err != nil {
			return 0, err
		}

		for _, tx := range res.Txs {
			for _, event := range tx.TxResult.Events {
				if event.Type == ophosttypes.EventTypeInitiateTokenDeposit {
					for _, attr := range event.Attributes {
						if attr.Key == ophosttypes.AttributeKeyBridgeId && attr.Value == strconv.FormatUint(bridgeId, 10) {
							return tx.Height, nil
						}
					}
				}
			}
		}

		if page*per_page >= res.TotalCount {
			break
		}
	}
	return 0, nil
}

func (b BaseHost) QueryBlock(ctx context.Context, height int64) (*coretypes.ResultBlock, error) {
	return b.node.GetRPCClient().Block(ctx, &height)
}

// OraclePriceHashWithProof contains oracle price hash and its proof
type OraclePriceHashWithProof struct {
	OraclePriceHash ophosttypes.OraclePriceHash
	Proof           []byte
	QueryHeight     uint64
}

// QueryOraclePriceHashWithProof queries L1 x/ophost module for oracle price hash along with proof
func (b BaseHost) QueryOraclePriceHashWithProof(ctx context.Context, height uint64) (*OraclePriceHashWithProof, error) {
	// query abci with prove=true from x/ophost module
	req := abci.RequestQuery{
		Data:   ophosttypes.OraclePriceHashPrefix,
		Path:   "/store/ophost/key",
		Height: int64(height),
		Prove:  true,
	}
	ctx, cancel := rpcclient.GetQueryContext(ctx, 0)
	defer cancel()

	res, err := b.node.GetRPCClient().QueryABCI(ctx, req)
	if err != nil {
		return nil, err
	}

	var oraclePriceHash ophosttypes.OraclePriceHash
	if err := oraclePriceHash.Unmarshal(res.GetValue()); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal oracle price hash")
	}
	if res.ProofOps == nil {
		return nil, errors.New("proof not available for oracle price hash query")
	}

	proofBytes, err := res.ProofOps.Marshal()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal proof")
	}

	return &OraclePriceHashWithProof{
		OraclePriceHash: oraclePriceHash,
		Proof:           proofBytes,
		QueryHeight:     height,
	}, nil
}

// QueryAllCurrencyPairs queries all available currency pairs from L1 Connect Oracle module
func (b BaseHost) QueryAllCurrencyPairs(ctx context.Context) ([]connecttypes.CurrencyPair, error) {
	req := &oracletypes.GetAllCurrencyPairsRequest{}
	ctx, cancel := rpcclient.GetQueryContext(ctx, 0)
	defer cancel()

	connectClient := oracletypes.NewQueryClient(b.node.GetRPCClient())
	res, err := connectClient.GetAllCurrencyPairs(ctx, req)
	if err != nil {
		return nil, err
	}

	return res.CurrencyPairs, nil
}

// QueryOraclePrices queries a single currency pair price from L1 Connect Oracle module
func (b BaseHost) QueryOraclePrices(ctx context.Context, currencyIds []string, height int64) ([]oracletypes.GetPriceResponse, error) {
	req := &oracletypes.GetPricesRequest{
		CurrencyPairIds: currencyIds,
	}
	ctx, cancel := rpcclient.GetQueryContext(ctx, height)
	defer cancel()

	connectClient := oracletypes.NewQueryClient(b.node.GetRPCClient())
	res, err := connectClient.GetPrices(ctx, req)
	if err != nil {
		return nil, err
	}

	return res.GetPrices(), nil
}

// QueryCommit queries the commit at a specific height from L1
func (b BaseHost) QueryCommit(ctx context.Context, height int64) (*coretypes.ResultCommit, error) {
	ctx, cancel := rpcclient.GetQueryContext(ctx, 0)
	defer cancel()

	return b.node.GetRPCClient().Commit(ctx, &height)
}

// QueryValidators queries all validators at a specific height from L1, along with pagination
func (b BaseHost) QueryValidators(ctx context.Context, height int64) ([]*cmtypes.Validator, error) {
	ctx, cancel := rpcclient.GetQueryContext(ctx, 0)
	defer cancel()

	page := 1
	perPage := 100

	validators := make([]*cmtypes.Validator, 0)
	for {
		result, err := b.node.GetRPCClient().Validators(ctx, &height, &page, &perPage)
		if err != nil {
			return nil, err
		}
		validators = append(validators, result.Validators...)
		page++
		if len(validators) >= result.Total {
			break
		}
	}

	return validators, nil
}

// QueryLatestHeight queries the latest block height from L1
func (b BaseHost) QueryLatestHeight(ctx context.Context) (int64, error) {
	ctx, cancel := rpcclient.GetQueryContext(ctx, 0)
	defer cancel()

	status, err := b.node.GetRPCClient().Status(ctx)
	if err != nil {
		return 0, err
	}

	return status.SyncInfo.LatestBlockHeight, nil
}
