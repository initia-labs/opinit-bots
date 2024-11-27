package child

import (
	"github.com/initia-labs/opinit-bots/node"
	"github.com/initia-labs/opinit-bots/node/broadcaster"
	nodetypes "github.com/initia-labs/opinit-bots/node/types"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/pkg/errors"
	"golang.org/x/exp/maps"
)

func (ch *Child) beginBlockHandler(ctx types.Context, args nodetypes.BeginBlockArgs) error {
	ch.EmptyMsgQueue()
	ch.EmptyProcessedMsgs()
	ch.stage.Reset()
	maps.Clear(ch.addressIndexMap)

	err := ch.prepareTree(args.Block.Header.Height)
	if err != nil {
		return errors.Wrap(err, "failed to prepare tree")
	}

	err = ch.prepareOutput(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to prepare output")
	}
	return nil
}

func (ch *Child) endBlockHandler(ctx types.Context, args nodetypes.EndBlockArgs) error {
	blockHeight := args.Block.Header.Height
	storageRoot, err := ch.handleTree(ctx, blockHeight, args.LatestHeight, args.BlockID, args.Block.Header)
	if err != nil {
		return errors.Wrap(err, "failed to handle tree")
	}

	if storageRoot != nil {
		workingTree, err := ch.WorkingTree()
		if err != nil {
			return errors.Wrap(err, "failed to get working tree")
		}
		err = ch.handleOutput(blockHeight, ch.Version(), args.BlockID, workingTree.Index, storageRoot)
		if err != nil {
			return errors.Wrap(err, "failed to handle output")
		}
	}

	// update the sync info
	err = node.SetSyncedHeight(ch.stage, args.Block.Header.Height)
	if err != nil {
		return errors.Wrap(err, "failed to set synced height")
	}

	// if has key, then process the messages
	if ch.host.HasBroadcaster() {
		ch.AppendProcessedMsgs(broadcaster.MsgsToProcessedMsgs(ch.GetMsgQueue())...)

		// save processed msgs to stage using host db
		err := ch.stage.ExecuteFnWithDB(ch.host.DB(), func() error {
			return broadcaster.SaveProcessedMsgsBatch(ch.stage, ch.host.Codec(), ch.GetProcessedMsgs())
		})
		if err != nil {
			return errors.Wrap(err, "failed to save processed msgs")
		}
	} else {
		ch.EmptyProcessedMsgs()
	}

	err = ch.stage.Commit()
	if err != nil {
		return errors.Wrap(err, "failed to commit stage")
	}

	ch.host.BroadcastProcessedMsgs(ch.GetProcessedMsgs()...)
	return nil
}
