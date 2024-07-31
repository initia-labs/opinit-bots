package types

type NodeConfig struct {
	RPC     string `json:"rpc"`
	ChainID string `json:"chain_id"`

	// Mnemonic is the mnemonic phrase for the bot account.
	//
	// If you don't want to use the mnemonic, you can leave it empty.
	// Then the bot will skip the tx submission.
	Account  string `json:"account"`
	GasPrice string `json:"gas_price"`
}
