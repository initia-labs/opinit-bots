package types

import (
	"fmt"

	comettypes "github.com/cometbft/cometbft/types"
)

func TxHash(txBytes []byte) string {
	return fmt.Sprintf("%X", comettypes.Tx(txBytes).Hash())
}
