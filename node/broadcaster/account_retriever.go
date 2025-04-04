package broadcaster

import (
	"fmt"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	grpctypes "github.com/cosmos/cosmos-sdk/types/grpc"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

var _ client.AccountRetriever = &BroadcasterAccount{}

// GetAccount queries for an account given an address and a block height. An
// error is returned if the query or decoding fails.
func (b *BroadcasterAccount) GetAccount(clientCtx client.Context, addr sdk.AccAddress) (client.Account, error) {
	account, _, err := b.GetAccountWithHeight(clientCtx, addr)
	return account, err
}

// GetAccountWithHeight queries for an account given an address. Returns the
// height of the query with the account. An error is returned if the query
// or decoding fails.
func (b *BroadcasterAccount) GetAccountWithHeight(clienCtx client.Context, addr sdk.AccAddress) (client.Account, int64, error) {
	var header metadata.MD
	queryClient := authtypes.NewQueryClient(b.rpcClient)
	res, err := queryClient.Account(clienCtx.CmdContext, &authtypes.QueryAccountRequest{Address: b.addressString}, grpc.Header(&header))
	if err != nil {
		return nil, 0, err
	}

	blockHeight := header.Get(grpctypes.GRPCBlockHeightHeader)
	if l := len(blockHeight); l != 1 {
		return nil, 0, fmt.Errorf("unexpected '%s' header length; got %d, expected: %d", grpctypes.GRPCBlockHeightHeader, l, 1)
	}

	nBlockHeight, err := strconv.Atoi(blockHeight[0])
	if err != nil {
		return nil, 0, fmt.Errorf("failed to parse block height: %w", err)
	}

	var acc sdk.AccountI
	if err := b.cdc.UnpackAny(res.Account, &acc); err != nil {
		return nil, 0, err
	}

	return acc, int64(nBlockHeight), nil
}

// EnsureExists returns an error if no account exists for the given address else nil.
func (b *BroadcasterAccount) EnsureExists(clientCtx client.Context, addr sdk.AccAddress) error {
	if _, err := b.GetAccount(clientCtx, addr); err != nil {
		return err
	}
	return nil
}

// GetAccountNumberSequence returns sequence and account number for the given address.
// It returns an error if the account couldn't be retrieved from the state.
func (b *BroadcasterAccount) GetAccountNumberSequence(clientCtx client.Context, addr sdk.AccAddress) (uint64, uint64, error) {
	acc, err := b.GetAccount(clientCtx, addr)
	if err != nil {
		return 0, 0, err
	}
	return acc.GetAccountNumber(), acc.GetSequence(), nil
}
