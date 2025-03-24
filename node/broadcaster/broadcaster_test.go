package broadcaster

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTxNotFoundRegex(t *testing.T) {
	errString := "error in json rpc client, with http response metadata: (Status: 200 OK, Protocol HTTP/1.1). RPC error -32603 - Internal error: tx (B6EBCDDA88B2758F72B36DB4BB4A3EABEF44FB127AD97E4A05535C9AE40947CA) not found"
	matches := txNotFoundRegex.FindStringSubmatch(errString)

	require.NotNil(t, matches)
}
