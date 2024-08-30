package child

import (
	"time"

	comettypes "github.com/cometbft/cometbft/types"
	challengertypes "github.com/initia-labs/opinit-bots/challenger/types"
)

func (ch *Child) oracleTxHandler(blockTime time.Time, l1BlockHeight uint64, oracleDataBytes comettypes.Tx) {
	checksum := challengertypes.OracleChecksum(oracleDataBytes)
	oracle := challengertypes.NewOracle(l1BlockHeight, checksum, blockTime)

	ch.elemQueue = append(ch.elemQueue, challengertypes.ChallengeElem{
		Node:  ch.NodeType(),
		Id:    oracle.L1Height,
		Event: oracle,
	})
}
