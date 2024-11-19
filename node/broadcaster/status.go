package broadcaster

import (
	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
)

func (b Broadcaster) GetStatus() btypes.BroadcasterStatus {
	return btypes.BroadcasterStatus{
		PendingTxs:     b.LenLocalPendingTx(),
		AccountsStatus: b.getAccountsStatus(),
	}
}

func (b Broadcaster) getAccountsStatus() []btypes.BroadcasterAccountStatus {
	accountsStatus := make([]btypes.BroadcasterAccountStatus, 0, len(b.accounts))
	for _, account := range b.accounts {
		accountsStatus = append(accountsStatus, btypes.BroadcasterAccountStatus{
			Address:  account.addressString,
			Sequence: account.Sequence(),
		})
	}
	return accountsStatus
}
