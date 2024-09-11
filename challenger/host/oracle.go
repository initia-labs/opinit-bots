package host

import (
	"time"

	comettypes "github.com/cometbft/cometbft/types"

	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
)

func (h *Host) oracleTxHandler(blockHeight uint64, blockTime time.Time, oracleDataBytes comettypes.Tx) {
	if !h.OracleEnabled() {
		return
	}
	checksum := challengertypes.OracleChecksum(oracleDataBytes)
	oracle := challengertypes.NewOracle(blockHeight, checksum, blockTime)

	h.eventQueue = append(h.eventQueue, oracle)
}
