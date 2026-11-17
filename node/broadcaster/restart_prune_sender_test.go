package broadcaster

import (
	"context"
	"sync"
	"testing"

	"github.com/initia-labs/opinit-bots/db"
	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// two accounts share one pending queue (bridge executor and oracle on an oracle-enabled rollup).
// a sequence recovery  for the oracle account must not drop the executor's saved transactions.
func TestRemovePendingTxsUntilKeepsOtherSenders(t *testing.T) {
	baseDB, err := db.NewMemDB()
	require.NoError(t, err)
	b := &Broadcaster{db: baseDB, pendingTxMu: &sync.Mutex{}}

	oracle, executor := "init1oracle", "init1executor"
	txs := []btypes.PendingTxInfo{
		{Sender: oracle, Sequence: 5000, TxHash: "A", Timestamp: 1, Save: false},
		{Sender: executor, Sequence: 12, TxHash: "B", Timestamp: 2, Save: true},
		{Sender: oracle, Sequence: 5001, TxHash: "C", Timestamp: 3, Save: false},
		{Sender: executor, Sequence: 13, TxHash: "D", Timestamp: 4, Save: true},
		{Sender: oracle, Sequence: 5005, TxHash: "E", Timestamp: 5, Save: false},
	}
	for _, tx := range txs {
		if tx.Save {
			require.NoError(t, SavePendingTx(baseDB, tx))
		}
	}
	b.pendingTxs = append(b.pendingTxs, txs...)

	ctx := types.NewContext(context.TODO(), zap.NewNop(), "")
	// the oracle account's on-chain sequence is 5003: its transactions up to 5002 are done
	require.NoError(t, b.RemovePendingTxsUntil(ctx, 5002))

	var left []string
	for _, tx := range b.pendingTxs {
		left = append(left, tx.TxHash)
	}
	require.Equal(t, []string{"B", "D", "E"}, left, "only the oracle's processed transactions leave the queue")

	saved, err := LoadPendingTxs(baseDB)
	require.NoError(t, err)
	require.Len(t, saved, 2, "the executor's saved transactions are still in the database")
}
