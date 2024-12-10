package executor

import (
	"github.com/initia-labs/opinit-bots/executor/batchsubmitter"
	"github.com/initia-labs/opinit-bots/executor/child"
	"github.com/initia-labs/opinit-bots/executor/host"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/pkg/errors"
)

type Status struct {
	BridgeId       uint64                `json:"bridge_id"`
	Host           host.Status           `json:"host,omitempty"`
	Child          child.Status          `json:"child,omitempty"`
	BatchSubmitter batchsubmitter.Status `json:"batch_submitter,omitempty"`
	DA             nodetypes.Status      `json:"da,omitempty"`
}

func (ex Executor) GetStatus() (Status, error) {
	var err error

	s := Status{}
	if ex.host != nil {
		s.BridgeId = ex.host.BridgeId()
		s.Host, err = ex.host.GetStatus()
		if err != nil {
			return Status{}, errors.Wrap(err, "failed to get host status")
		}
	}
	if ex.child != nil {
		s.Child, err = ex.child.GetStatus()
		if err != nil {
			return Status{}, errors.Wrap(err, "failed to get child status")
		}
	}
	if ex.batchSubmitter != nil {
		s.BatchSubmitter, err = ex.batchSubmitter.GetStatus()
		if err != nil {
			return Status{}, errors.Wrap(err, "failed to get batch status")
		}
		if ex.batchSubmitter.DA() != nil {
			s.DA, err = ex.batchSubmitter.DA().GetNodeStatus()
			if err != nil {
				return Status{}, errors.Wrap(err, "failed to get DA status")
			}
		}
	}
	return s, nil
}
