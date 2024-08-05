package broadcaster

import (
	"context"
	"fmt"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	grpctypes "github.com/cosmos/cosmos-sdk/types/grpc"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/initia-labs/opinit-bots-go/keys"
)

var _ client.AccountRetriever = &Broadcaster{}

func (b *Broadcaster) loadAccount() error {
	account, err := b.GetAccount(b.getClientCtx(), b.keyAddress)
	if err != nil {
		return err
	}
	b.txf = b.txf.WithAccountNumber(account.GetAccountNumber()).WithSequence(account.GetSequence())
	return nil
}

func (b Broadcaster) GetAddress() sdk.AccAddress {
	return b.keyAddress
}

func (b Broadcaster) GetAddressString() (string, error) {
	return keys.EncodeBech32AccAddr(b.keyAddress, b.cfg.Bech32Prefix)
}

// GetAccount queries for an account given an address and a block height. An
// error is returned if the query or decoding fails.
func (b *Broadcaster) GetAccount(clientCtx client.Context, addr sdk.AccAddress) (client.Account, error) {
	account, _, err := b.GetAccountWithHeight(clientCtx, addr)
	return account, err
}

// GetAccountWithHeight queries for an account given an address. Returns the
// height of the query with the account. An error is returned if the query
// or decoding fails.
func (b *Broadcaster) GetAccountWithHeight(_ client.Context, addr sdk.AccAddress) (client.Account, int64, error) {
	var header metadata.MD
	address, err := keys.EncodeBech32AccAddr(addr, b.cfg.Bech32Prefix)
	if err != nil {
		return nil, 0, err
	}

	queryClient := authtypes.NewQueryClient(b.rpcClient)
	res, err := queryClient.Account(context.Background(), &authtypes.QueryAccountRequest{Address: address}, grpc.Header(&header))
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

	//nolint:staticcheck
	var acc authtypes.AccountI
	if err := b.cdc.UnpackAny(res.Account, &acc); err != nil {
		return nil, 0, err
	}

	return acc, int64(nBlockHeight), nil
}

// EnsureExists returns an error if no account exists for the given address else nil.
func (b *Broadcaster) EnsureExists(clientCtx client.Context, addr sdk.AccAddress) error {
	if _, err := b.GetAccount(clientCtx, addr); err != nil {
		return err
	}
	return nil
}

// GetAccountNumberSequence returns sequence and account number for the given address.
// It returns an error if the account couldn't be retrieved from the state.
func (b *Broadcaster) GetAccountNumberSequence(clientCtx client.Context, addr sdk.AccAddress) (uint64, uint64, error) {
	acc, err := b.GetAccount(clientCtx, addr)
	if err != nil {
		return 0, 0, err
	}
	return acc.GetAccountNumber(), acc.GetSequence(), nil
}
