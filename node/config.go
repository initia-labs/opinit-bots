package node

type NodeConfig struct {
	RPC      string `json:"rpc"`
	ChainID  string `json:"chain_id"`
	Mnemonic string `json:"mnemonic"`
	GasPrice string `json:"gas_price"`
}
