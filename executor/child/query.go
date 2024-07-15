package child

import (
	"encoding/json"
	"errors"

	opchildtypes "github.com/initia-labs/OPinit/x/opchild/types"
	executortypes "github.com/initia-labs/opinit-bots-go/executor/types"
	"github.com/initia-labs/opinit-bots-go/node"
)

func (ch Child) GetAddressStr() (string, error) {
	addr := ch.node.GetAddress()
	if addr == nil {
		return "", errors.New("nil address")
	}
	return ch.ac.BytesToString(addr)
}

func (ch Child) QueryBridgeInfo() (opchildtypes.BridgeInfo, error) {
	req := &opchildtypes.QueryBridgeInfoRequest{}
	ctx := node.GetQueryContext(0)
	res, err := ch.opchildQueryClient.BridgeInfo(ctx, req)
	if err != nil {
		return opchildtypes.BridgeInfo{}, err
	}
	return res.BridgeInfo, nil
}

func (ch Child) QueryNextL2Sequence(height uint64) (uint64, error) {
	req := &opchildtypes.QueryNextL2SequenceRequest{}
	ctx := node.GetQueryContext(height)
	res, err := ch.opchildQueryClient.NextL2Sequence(ctx, req)
	if err != nil {
		return 0, err
	}
	return res.NextL2Sequence, nil
}

func (ch Child) QueryProofs(sequence uint64) (executortypes.QueryProofsResponse, error) {
	proofs, outputIndex, outputRoot, extraData, err := ch.mk.GetProofs(sequence)
	if err != nil {
		return executortypes.QueryProofsResponse{}, err
	}

	data := executortypes.TreeExtraData{}
	err = json.Unmarshal(extraData, &data)
	if err != nil {
		return executortypes.QueryProofsResponse{}, err
	}

	return executortypes.QueryProofsResponse{
		BridgeId:         ch.BridgeId(),
		Sequence:         sequence,
		Version:          []byte{ch.version},
		WithdrawalProofs: proofs,
		OutputIndex:      outputIndex,
		StorageRoot:      outputRoot,
		BlockNumber:      data.BlockNumber,
		LatestBlockHash:  data.BlockHash,
	}, nil
}
