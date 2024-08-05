package host

// import (
// 	"testing"

// 	"github.com/initia-labs/opinit-bots-go/db"
// 	"github.com/initia-labs/opinit-bots-go/node/types"
// 	"github.com/stretchr/testify/require"
// 	"go.uber.org/zap"
// 	"go.uber.org/zap/zaptest/observer"
// )

// func defaultConfig() types.NodeConfig {
// 	return types.NodeConfig{
// 		RPC:     "http://localhost:26657",
// 		ChainID: "testnet-1",
// 		Account: "test-acc",
// 		GasPrice: "0.15uinit",
// 	}
// }

// func logCapture() (*zap.Logger, *observer.ObservedLogs) {
// 	core, logs := observer.New(zap.InfoLevel)
// 	return zap.New(core), logs
// }

// func Test_handleProposeOutput(t *testing.T) {
// 	db, err := db.NewDB(t.TempDir())
// 	require.NoError(t, err)

// 	defer db.Close()

// 	logger, logs := logCapture()
// 	host := NewHost(1, true, defaultConfig(), db, logger, t.TempDir(), "")
// }
