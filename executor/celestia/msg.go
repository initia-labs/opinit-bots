package celestia

import (
	"errors"

	"github.com/cometbft/cometbft/crypto/merkle"

	sdk "github.com/cosmos/cosmos-sdk/types"

	inclusion "github.com/celestiaorg/go-square/v2/inclusion"
	sh "github.com/celestiaorg/go-square/v2/share"

	"github.com/initia-labs/opinit-bots/types"
	celestiatypes "github.com/initia-labs/opinit-bots/types/celestia"
)

func (c Celestia) CreateBatchMsg(rawBlob []byte) (sdk.Msg, string, error) {
	submitter, err := c.BaseAccountAddress()
	if err != nil {
		if errors.Is(err, types.ErrKeyNotSet) {
			return nil, "", nil
		}
		return nil, "", err
	}
	blob, err := sh.NewV0Blob(c.namespace, rawBlob)
	if err != nil {
		return nil, "", err
	}
	commitment, err := inclusion.CreateCommitment(blob,
		merkle.HashFromByteSlices,
		// https://github.com/celestiaorg/celestia-app/blob/4f4d0f7ff1a43b62b232726e52d1793616423df7/pkg/appconsts/v1/app_consts.go#L6
		64,
	)
	if err != nil {
		return nil, "", err
	}

	dataLength, err := types.SafeIntToUint32(len(blob.Data()))
	if err != nil {
		return nil, "", err
	}

	return &celestiatypes.MsgPayForBlobsWithBlob{
		MsgPayForBlobs: &celestiatypes.MsgPayForBlobs{
			Signer:           submitter,
			Namespaces:       [][]byte{c.namespace.Bytes()},
			ShareCommitments: [][]byte{commitment},
			BlobSizes:        []uint32{dataLength},
			ShareVersions:    []uint32{uint32(blob.ShareVersion())},
		},
		Blob: &celestiatypes.Blob{
			NamespaceId:      blob.Namespace().ID(),
			Data:             blob.Data(),
			ShareVersion:     uint32(blob.ShareVersion()),
			NamespaceVersion: uint32(blob.Namespace().Version()),
		},
	}, submitter, nil
}
