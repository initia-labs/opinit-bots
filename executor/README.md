# Executor

The Executor is responsible for relaying L1 token deposit transactions to L2 and periodically submitting L2 output roots to L1.

## Config

To configure the Executor, fill in the values in the `~/.opinit/executor.json` file.

```json
{
  // Version is the version used to build output root.
  "version": 1,

  // ListenAddress is the address to listen for incoming requests.
  "listen_address": "tcp://localhost:3000",

  "l1_rpc_address": "tcp://localhost:26657",
  "l2_rpc_address": "tcp://localhost:27657",

  "l1_gas_price": "0.15uinit",
  "l2_gas_price": "",

  "l1_chain_id": "testnet-l1-1",
  "l2_chain_id": "testnet-l2-1",

  // OutputSubmitterMnemonic is the mnemonic phrase for the output submitter,
  // which is used to relay the output transaction from l2 to l1.
  //
  // If you don't want to use the output submitter feature, you can leave it empty.
  "output_submitter_mnemonic": "",

  // BridgeExecutorMnemonic is the mnemonic phrase for the bridge executor,
  // which is used to relay initiate token bridge transaction from l1 to l2.
  //
  // If you don't want to use the bridge executor feature, you can leave it empty.
  "bridge_executor_mnemonic": "",

  // RelayOracle is the flag to enable the oracle relay feature.
  "relay_oracle": false
}
```
