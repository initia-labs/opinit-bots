package executor

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (ex *Executor) addChildMsg(msg sdk.Msg) {
	ex.childMsgQueue = append(ex.childMsgQueue, msg)
}

func (ex *Executor) deleteChildMsgQueue() {
	ex.childMsgQueue = ex.childMsgQueue[:0]
}

func (ex *Executor) deleteHostProcessedMsgs() {
	ex.hostProcessedMsgs = ex.hostProcessedMsgs[:0]
}
