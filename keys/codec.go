package keys

import (
	"cosmossdk.io/errors"
	"cosmossdk.io/x/tx/signing"
	"github.com/cosmos/gogoproto/proto"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codecaddress "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
)

type RegisterInterfaces func(registry codectypes.InterfaceRegistry)

func CreateCodec(registerFns []RegisterInterfaces) (codectypes.InterfaceRegistry, codec.Codec, client.TxConfig, error) {
	interfaceRegistry, err := codectypes.NewInterfaceRegistryWithOptions(codectypes.InterfaceRegistryOptions{
		ProtoFiles: proto.HybridResolver,
		SigningOptions: signing.Options{
			AddressCodec:          codecaddress.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix()),
			ValidatorAddressCodec: codecaddress.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()),
		},
	})
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to create interface registry")
	}
	appCodec := codec.NewProtoCodec(interfaceRegistry)
	interfaceRegistry.RegisterImplementations((*cryptotypes.PubKey)(nil), &PubKey{})
	interfaceRegistry.RegisterImplementations((*cryptotypes.PrivKey)(nil), &PrivKey{})
	std.RegisterInterfaces(interfaceRegistry)
	for _, fn := range registerFns {
		fn(interfaceRegistry)
	}

	txConfig := tx.NewTxConfig(appCodec, tx.DefaultSignModes)
	return interfaceRegistry, appCodec, txConfig, nil
}
