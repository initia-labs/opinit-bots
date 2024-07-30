package node

import (
	"context"
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	grpctypes "github.com/cosmos/cosmos-sdk/types/grpc"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var _ client.AccountRetriever = &Node{}

func (n *Node) loadAccount() error {
	account, err := n.GetAccount(n.getClientCtx(), n.keyAddress)
	if err != nil {
		return err
	}
	n.txf = n.txf.WithAccountNumber(account.GetAccountNumber())
	n.txf = n.txf.WithSequence(account.GetSequence())
	return nil
}

func (n Node) GetAddress() sdk.AccAddress {
	return n.keyAddress
}

func (n Node) GetAddressString() (string, error) {
	return n.EncodeBech32AccAddr(n.keyAddress)
}

// GetAccount queries for an account given an address and a block height. An
// error is returned if the query or decoding fails.
func (n *Node) GetAccount(clientCtx client.Context, addr sdk.AccAddress) (client.Account, error) {
	account, _, err := n.GetAccountWithHeight(clientCtx, addr)
	return account, err
}

// GetAccountWithHeight queries for an account given an address. Returns the
// height of the query with the account. An error is returned if the query
// or decoding fails.
func (n *Node) GetAccountWithHeight(_ client.Context, addr sdk.AccAddress) (client.Account, int64, error) {
	var header metadata.MD
	address, err := n.EncodeBech32AccAddr(addr)
	if err != nil {
		return nil, 0, err
	}

	queryClient := authtypes.NewQueryClient(n)
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
	if err := n.cdc.UnpackAny(res.Account, &acc); err != nil {
		return nil, 0, err
	}

	return acc, int64(nBlockHeight), nil
}

// EnsureExists returns an error if no account exists for the given address else nil.
func (n *Node) EnsureExists(clientCtx client.Context, addr sdk.AccAddress) error {
	if _, err := n.GetAccount(clientCtx, addr); err != nil {
		return err
	}
	return nil
}

// GetAccountNumberSequence returns sequence and account number for the given address.
// It returns an error if the account couldn't be retrieved from the state.
func (n *Node) GetAccountNumberSequence(clientCtx client.Context, addr sdk.AccAddress) (uint64, uint64, error) {
	acc, err := n.GetAccount(clientCtx, addr)
	if err != nil {
		return 0, 0, err
	}
	return acc.GetAccountNumber(), acc.GetSequence(), nil
}

func (n *Node) EncodeBech32AccAddr(addr sdk.AccAddress) (string, error) {
	return bech32.ConvertAndEncode(n.bech32Prefix, addr)
}

func (n *Node) DecodeBech32AccAddr(addr string) (sdk.AccAddress, error) {
	_, bz, err := bech32.DecodeAndConvert(addr)
	return bz, err
}
