package mockclient

import (
	"context"
	"errors"

	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	jsonrpcclient "github.com/cometbft/cometbft/rpc/jsonrpc/client"
	clienttypes "github.com/initia-labs/opinit-bots/client/types"
)

type MockCaller struct {
	latestHeight int64
	resultStatus ctypes.ResultStatus
	rawCommits   map[int64][]byte
}

func NewMockCaller() *MockCaller {
	return &MockCaller{
		rawCommits: make(map[int64][]byte),
	}
}

var _ jsonrpcclient.Caller = (*MockCaller)(nil)

func (m *MockCaller) Call(ctx context.Context, method string, params map[string]interface{}, result interface{}) (interface{}, error) {
	switch method {
	case "status":
		return m.status(params, result.(*ctypes.ResultStatus))
	case "raw_commit":
		return m.rawCommit(params, result.(*clienttypes.ResultRawCommit))
	}
	return nil, errors.New("not supported method")
}

func (m *MockCaller) SetLatestHeight(height int64) {
	m.latestHeight = height
}

func (m *MockCaller) SetResultStatus(result ctypes.ResultStatus) {
	m.resultStatus = result
}

func (m *MockCaller) status(_ map[string]interface{}, result *ctypes.ResultStatus) (interface{}, error) {
	*result = m.resultStatus
	return nil, nil
}

func (m *MockCaller) SetRawCommit(height int64, commitBytes []byte) {
	m.rawCommits[height] = commitBytes
}

func (m *MockCaller) rawCommit(params map[string]interface{}, result *clienttypes.ResultRawCommit) (interface{}, error) {
	h := params["height"].(*int64)
	height := m.latestHeight
	if h != nil {
		height = *h
	}

	commitBytes, ok := m.rawCommits[height]
	if !ok {
		return nil, errors.New("commit not found")
	}
	*result = clienttypes.ResultRawCommit{
		Commit: commitBytes,
	}
	return nil, nil
}
