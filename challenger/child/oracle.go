package child

import (
	"time"

	comettypes "github.com/cometbft/cometbft/types"
	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
	"github.com/initia-labs/opinit-bots/types"
	"go.uber.org/zap"
)

func (ch *Child) oracleTxHandler(ctx types.Context, blockTime time.Time, sender string, l1BlockHeight int64, oracleDataBytes comettypes.Tx) {
	checksum := challengertypes.OracleChecksum(oracleDataBytes)
	oracle := challengertypes.NewOracle(l1BlockHeight, checksum, blockTime)
	ch.eventQueue = append(ch.eventQueue, oracle)
	ch.lastUpdatedOracleL1Height = l1BlockHeight

	ctx.Logger().Info("update oracle",
		zap.Int64("l1_blockHeight", l1BlockHeight),
		zap.String("from", sender),
	)
}
