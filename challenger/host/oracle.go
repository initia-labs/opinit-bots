package host

import (
	"context"
	"time"

	comettypes "github.com/cometbft/cometbft/types"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	hostprovider "github.com/initia-labs/opinit-bots/provider/host"
	"github.com/initia-labs/opinit-bots/types"
)

func (h *Host) oracleTxHandler(blockHeight int64, blockTime time.Time, oracleDataBytes comettypes.Tx) {
	if !h.OracleEnabled() {
		return
	}
	checksum := challengertypes.OracleChecksum(oracleDataBytes)
	oracle := challengertypes.NewOracle(blockHeight, checksum, blockTime)

	h.eventQueue = append(h.eventQueue, oracle)
}

func (h *Host) oracleRelayHandler(ctx context.Context, blockHeight int64, blockTime time.Time) error {
	if !h.OracleEnabled() {
		return nil
	}

	// query OraclePriceHash from x/ophost module state at previous height
	// state at height H is available after block H is committed
	queryHeight := uint64(blockHeight - 1)
	if blockHeight <= 1 {
		return nil
	}

	oraclePriceHashData, err := h.QueryOraclePriceHashWithProof(ctx, queryHeight)
	if err != nil {
		// OraclePriceHash may not exist yet, skip silently
		return nil
	}

	if oraclePriceHashData == nil || len(oraclePriceHashData.OraclePriceHash.Hash) == 0 {
		return nil
	}

	oracleRelay := challengertypes.NewOracleRelay(
		types.MustUint64ToInt64(oraclePriceHashData.OraclePriceHash.L1BlockHeight),
		oraclePriceHashData.OraclePriceHash.Hash,
		blockTime,
	)

	h.eventQueue = append(h.eventQueue, oracleRelay)
	return nil
}

func (h *Host) updateOracleConfigHandler(ctx types.Context, args nodetypes.EventHandlerArgs) error {
	bridgeId, oracleEnabled, err := hostprovider.ParseMsgUpdateOracleConfig(args.EventAttributes)
	if err != nil {
		return errors.Wrap(err, "failed to parse update oracle config event")
	}
	if bridgeId != h.BridgeId() {
		return nil
	}

	ctx.Logger().Info("update oracle config",
		zap.Uint64("bridge_id", bridgeId),
		zap.Bool("oracle_enabled", oracleEnabled),
	)

	h.UpdateOracleEnabled(oracleEnabled)
	return nil
}
