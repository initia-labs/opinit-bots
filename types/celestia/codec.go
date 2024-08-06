package celestia

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RegisterInterfaces register the Ethermint key concrete types.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil), &MsgPayForBlobs{}, &MsgPayForBlobsWithBlob{})
}

func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgPayForBlobs{}, "/celestia.blob.v1.MsgPayForBlobs", nil)
	cdc.RegisterConcrete(&MsgPayForBlobsWithBlob{}, "/celestia.blob.v1.MsgPayForBlobsWithBlob", nil)
}
