package broadcaster

import (
	"fmt"
	"sync"

	"github.com/initia-labs/opinit-bots/keys"
	btypes "github.com/initia-labs/opinit-bots/node/broadcaster/types"
	"github.com/initia-labs/opinit-bots/node/rpcclient"
	"github.com/initia-labs/opinit-bots/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/hd"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func NewTestBroadcaster(cdc codec.Codec, db types.DB, rpcClient *rpcclient.RPCClient, txConfig client.TxConfig, prefix string, numAccounts int) (*Broadcaster, error) {
	b := &Broadcaster{
		cdc:       cdc,
		db:        db,
		rpcClient: rpcClient,

		txConfig:          txConfig,
		accounts:          make([]*BroadcasterAccount, 0),
		addressAccountMap: make(map[string]int),
		accountMu:         &sync.Mutex{},

		txChannel:        make(chan btypes.ProcessedMsgs),
		txChannelStopped: make(chan struct{}),

		pendingTxMu:               &sync.Mutex{},
		pendingTxs:                make([]btypes.PendingTxInfo, 0),
		pendingProcessedMsgsBatch: make([]btypes.ProcessedMsgs, 0),
	}

	for i := 0; i < numAccounts; i++ {
		keybase, err := keys.GetKeyBase("", "", cdc, nil)
		if err != nil {
			return nil, err
		}

		mnemonic, err := keys.CreateMnemonic()
		if err != nil {
			return nil, err
		}
		keyName := fmt.Sprintf("%d", i)
		account, err := keybase.NewAccount(keyName, mnemonic, "", hd.CreateHDPath(sdk.CoinType, 0, 0).String(), hd.Secp256k1)
		if err != nil {
			return nil, err
		}
		addr, err := account.GetAddress()
		if err != nil {
			return nil, err
		}
		addrString, err := keys.EncodeBech32AccAddr(addr, prefix)
		if err != nil {
			return nil, err
		}

		broadcasterAccount := &BroadcasterAccount{
			cdc:           cdc,
			txConfig:      txConfig,
			rpcClient:     rpcClient,
			keyName:       keyName,
			keyBase:       keybase,
			keyringRecord: account,
			address:       addr,
			addressString: addrString,
		}
		b.accounts = append(b.accounts, broadcasterAccount)
	}
	return b, nil
}
