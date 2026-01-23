package oraclerelay

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	cmtypes "github.com/cometbft/cometbft/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ibctmlightclients "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	connecttypes "github.com/skip-mev/connect/v2/pkg/types"
	oracletypes "github.com/skip-mev/connect/v2/x/oracle/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	ophosttypes "github.com/initia-labs/OPinit/x/ophost/types"

	executortypes "github.com/initia-labs/opinit-bots/executor/types"
	hostprovider "github.com/initia-labs/opinit-bots/provider/host"
	"github.com/initia-labs/opinit-bots/types"
)

// hostNode defines the interface for L1 operations needed by oracle relay
type hostNode interface {
	ChainId() string
	BridgeInfo() ophosttypes.QueryBridgeResponse
	OracleEnabled() bool
	QueryOraclePriceHashWithProof(context.Context, uint64) (*hostprovider.OraclePriceHashWithProof, error)
	QueryAllCurrencyPairs(context.Context) ([]connecttypes.CurrencyPair, error)
	QueryOraclePrices(context.Context, []string, int64) ([]oracletypes.GetPriceResponse, error)
	QueryCommit(context.Context, int64) (*coretypes.ResultCommit, error)
	QueryValidators(context.Context, int64) ([]*cmtypes.Validator, error)
	QueryLatestHeight(context.Context) (int64, error)
	QueryBlock(context.Context, int64) (*coretypes.ResultBlock, error)
}

// childNode defines the interface for L2 operations needed by oracle relay
type childNode interface {
	BroadcastMsgs([]sdk.Msg, string)
	QueryL1ClientID(context.Context) (string, error)
	QueryLatestRevisionHeight(context.Context, string) (uint64, error)
	BaseAccountAddressString() (string, error)
}

// OracleRelay handles batched oracle price relaying from L1 to L2
type OracleRelay struct {
	version uint8

	host   hostNode
	child  childNode
	cfg    executortypes.OracleRelayConfig
	sender string // sender address for relay messages

	// status info
	statusMu            sync.RWMutex
	lastRelayedL1Height uint64
	lastRelayedTime     time.Time
}

// NewOracleRelayV1 creates a new oracle relay handler
func NewOracleRelayV1(cfg executortypes.OracleRelayConfig) *OracleRelay {
	return &OracleRelay{
		version: 1,
		cfg:     cfg,
	}
}

// Initialize initializes the oracle relay with host and child dependencies
func (or *OracleRelay) Initialize(host hostNode, child childNode, sender string) error {
	or.host = host
	or.child = child
	or.sender = sender
	return nil
}

// Start starts the oracle relay loop
func (or *OracleRelay) Start(ctx types.Context) error {
	if !or.cfg.Enable {
		ctx.Logger().Info("oracle relay is disabled")
		return nil
	}

	ctx.Logger().Info("starting oracle relay handler",
		zap.Int64("interval", or.cfg.Interval),
		zap.Strings("currency_pairs", or.cfg.CurrencyPairs),
	)

	ticker := time.NewTicker(time.Duration(or.cfg.Interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			ctx.Logger().Info("oracle relay handler stopped")
			return nil
		case <-ticker.C:
			if err := or.relayOnce(ctx); err != nil {
				ctx.Logger().Error("oracle relay failed", zap.Error(err))
			}
		}
	}
}

// relayOnce attempts to relay oracle data once
func (or *OracleRelay) relayOnce(ctx types.Context) error {
	// check first if oracle is enabled
	if !or.host.OracleEnabled() {
		ctx.Logger().Debug("oracle is disabled for this bridge, skipping relay")
		return nil
	}

	bridgeInfo := or.host.BridgeInfo()
	l1ClientID, err := or.child.QueryL1ClientID(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get L1 client ID")
	}

	trustedHeight, err := or.child.QueryLatestRevisionHeight(ctx, l1ClientID)
	if err != nil {
		return errors.Wrap(err, "failed to get latest consensus height")
	}
	if trustedHeight == 0 {
		return errors.New("no consensus states available")
	}

	targetHeight, err := or.host.QueryLatestHeight(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to query latest L1 height")
	}
	queryHeight := uint64(targetHeight - 1) // query at height-1 to verify against height

	// skip if we already relayed this L1 height (no new oracle data)
	if queryHeight <= or.GetLastRelayedL1Height() {
		ctx.Logger().Debug("skipping relay, L1 height unchanged",
			zap.Uint64("query_height", queryHeight),
			zap.Uint64("last_relayed_l1_height", or.GetLastRelayedL1Height()),
		)
		return nil
	}

	ctx.Logger().Debug("querying oracle data",
		zap.Int64("target_height", targetHeight),
		zap.Uint64("query_height", queryHeight),
		zap.Uint64("trusted_height", trustedHeight),
	)

	revision, err := or.parseRevisionFromChainID(or.host.ChainId())
	if err != nil {
		return errors.Wrap(err, "failed to parse oracle relay revision")
	}

	var msgs []sdk.Msg
	header, err := or.buildIBCHeader(ctx, targetHeight, trustedHeight)
	if err != nil {
		return errors.Wrap(err, "failed to build IBC header")
	}

	headerAny, err := codectypes.NewAnyWithValue(header)
	if err != nil {
		return errors.Wrap(err, "failed to pack IBC header")
	}

	msgUpdateClient := &clienttypes.MsgUpdateClient{
		ClientId:      l1ClientID,
		ClientMessage: headerAny,
		Signer:        or.sender,
	}
	msgs = append(msgs, msgUpdateClient)

	ctx.Logger().Debug("adding MsgUpdateClient to transaction",
		zap.Int64("target_height", targetHeight),
		zap.Uint64("trusted_height", trustedHeight),
	)

	providerOracleData, err := or.host.QueryOraclePriceHashWithProof(ctx, queryHeight)
	if err != nil {
		return errors.Wrap(err, "failed to query oracle price hash")
	}

	var currencyPairs []connecttypes.CurrencyPair
	if len(or.cfg.CurrencyPairs) == 0 {
		currencyPairs, err = or.host.QueryAllCurrencyPairs(ctx)
		if err != nil {
			return errors.Wrap(err, "failed to query currency pairs")
		}
	} else {
		for _, cp := range or.cfg.CurrencyPairs {
			parts := strings.Split(cp, "/")
			if len(parts) == 2 {
				currencyPairs = append(currencyPairs, connecttypes.CurrencyPair{
					Base:  parts[0],
					Quote: parts[1],
				})
			}
		}
	}

	prices, err := or.queryAllOraclePrices(ctx, currencyPairs, int64(queryHeight))
	if err != nil {
		return errors.Wrap(err, "failed to query oracle prices")
	}
	if len(prices) == 0 {
		return errors.New("no oracle prices retrieved")
	}

	executorAddr, err := or.child.BaseAccountAddressString()
	if err != nil {
		return errors.Wrap(err, "failed to get executor address")
	}

	msgRelayOracleData := &opchildtypes.MsgRelayOracleData{
		Sender: executorAddr,
		OracleData: opchildtypes.OracleData{
			BridgeId:        bridgeInfo.BridgeId,
			OraclePriceHash: providerOracleData.OraclePriceHash.Hash,
			Prices:          prices,
			L1BlockHeight:   providerOracleData.OraclePriceHash.L1BlockHeight,
			L1BlockTime:     providerOracleData.OraclePriceHash.L1BlockTime,
			Proof:           providerOracleData.Proof,
			ProofHeight:     clienttypes.NewHeight(revision, uint64(targetHeight)),
		},
	}

	senderAddr, err := sdk.AccAddressFromBech32(or.sender)
	if err != nil {
		return errors.Wrap(err, "failed to parse sender address")
	}
	msgExec := authz.NewMsgExec(senderAddr, []sdk.Msg{msgRelayOracleData})
	msgs = append(msgs, &msgExec)

	or.child.BroadcastMsgs(msgs, or.sender)

	or.SetLastRelayedL1Height(providerOracleData.OraclePriceHash.L1BlockHeight)
	or.SetLastRelayedTime(time.Now())

	ctx.Logger().Info("successfully broadcasted oracle relay message",
		zap.Uint64("l1_height", providerOracleData.OraclePriceHash.L1BlockHeight),
		zap.Int("num_prices", len(prices)),
	)

	return nil
}

// queryAllOraclePrices queries all currency pair prices from L1 and transforms them to executor types
func (or *OracleRelay) queryAllOraclePrices(ctx types.Context, currencyPairs []connecttypes.CurrencyPair, height int64) ([]opchildtypes.OraclePriceData, error) {
	currencyPairIds := make([]string, len(currencyPairs))
	for idx, cp := range currencyPairs {
		currencyPairIds[idx] = cp.String()
	}

	prices, err := or.host.QueryOraclePrices(ctx, currencyPairIds, height)
	if err != nil {
		ctx.Logger().Error("failed to query prices", zap.Error(err))
		return nil, err
	}

	if len(prices) != len(currencyPairIds) {
		return nil, fmt.Errorf("oracle price count mismatch: got %d, expected %d",
			len(prices), len(currencyPairIds))
	}

	priceDatas := make([]opchildtypes.OraclePriceData, len(prices))

	for idx, price := range prices {
		quotePrice := price.GetPrice()
		priceDatas[idx] = opchildtypes.OraclePriceData{
			CurrencyPair:   currencyPairIds[idx],
			Price:          quotePrice.Price.String(),
			Decimals:       price.Decimals,
			Nonce:          price.Nonce,
			CurrencyPairId: price.Id,
			Timestamp:      quotePrice.GetBlockTimestamp().UnixNano(),
		}
	}

	return priceDatas, nil
}

// parseRevisionFromChainID extracts the revision number from chain ID, e.g. "interwoven-1" -> 1
func (or *OracleRelay) parseRevisionFromChainID(chainID string) (uint64, error) {
	parts := strings.Split(chainID, "-")
	if len(parts) > 0 {
		if rev, err := strconv.ParseUint(parts[len(parts)-1], 10, 64); err == nil {
			return rev, nil
		}
	}
	return 0, fmt.Errorf("cannot find revision from chain id %s", chainID)
}

// SetLastRelayedL1Height sets the last relayed L1 height
func (or *OracleRelay) SetLastRelayedL1Height(height uint64) {
	or.statusMu.Lock()
	defer or.statusMu.Unlock()
	or.lastRelayedL1Height = height
}

// GetLastRelayedL1Height returns the last relayed L1 height
func (or *OracleRelay) GetLastRelayedL1Height() uint64 {
	or.statusMu.RLock()
	defer or.statusMu.RUnlock()
	return or.lastRelayedL1Height
}

// SetLastRelayedTime sets the last relay time
func (or *OracleRelay) SetLastRelayedTime(t time.Time) {
	or.statusMu.Lock()
	defer or.statusMu.Unlock()
	or.lastRelayedTime = t
}

// GetLastRelayedTime returns the last relay time
func (or *OracleRelay) GetLastRelayedTime() time.Time {
	or.statusMu.RLock()
	defer or.statusMu.RUnlock()
	return or.lastRelayedTime
}

// buildIBCHeader constructs an IBC Tendermint header for client update
func (or *OracleRelay) buildIBCHeader(ctx types.Context, targetHeight int64, trustedHeight uint64) (*ibctmlightclients.Header, error) {
	commit, err := or.host.QueryCommit(ctx, targetHeight)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query commit")
	}

	validators, err := or.host.QueryValidators(ctx, targetHeight)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query validators at target height")
	}

	trustedValidators, err := or.host.QueryValidators(ctx, int64(trustedHeight))
	if err != nil {
		return nil, errors.Wrap(err, "failed to query validators at trusted height")
	}

	trustedBlock, err := or.host.QueryBlock(ctx, int64(trustedHeight))
	if err != nil {
		return nil, errors.Wrap(err, "failed to query block at trusted height")
	}

	validatorSet, err := or.toProtoValidatorSet(validators, commit.ProposerAddress)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert validator set")
	}

	trustedValidatorSet, err := or.toProtoValidatorSet(trustedValidators, trustedBlock.Block.ProposerAddress)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert trusted validator set")
	}

	revision, err := or.parseRevisionFromChainID(or.host.ChainId())
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse revision from chain ID")
	}

	return &ibctmlightclients.Header{
		SignedHeader:      commit.ToProto(),
		ValidatorSet:      validatorSet,
		TrustedHeight:     clienttypes.NewHeight(revision, trustedHeight),
		TrustedValidators: trustedValidatorSet,
	}, nil
}

// toProtoValidatorSet converts validators to proto format with optional proposer
func (or *OracleRelay) toProtoValidatorSet(validators []*cmtypes.Validator, proposerAddress []byte) (*cmtproto.ValidatorSet, error) {
	protoValidators := make([]*cmtproto.Validator, 0, len(validators))
	var proposer *cmtproto.Validator
	totalVotingPower := int64(0)

	for _, val := range validators {
		protoVal, err := val.ToProto()
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert validator to proto")
		}
		protoVal.ProposerPriority = 0
		totalVotingPower += protoVal.VotingPower
		protoValidators = append(protoValidators, protoVal)

		if proposerAddress != nil && bytes.Equal(protoVal.Address, proposerAddress) {
			proposer = protoVal
		}
	}

	return &cmtproto.ValidatorSet{
		Validators:       protoValidators,
		Proposer:         proposer,
		TotalVotingPower: totalVotingPower,
	}, nil
}
