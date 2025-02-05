# UpdateBatchInfo
Follow the steps below.
### Step 1. Register the key for the new batch submitter address.
`opinitd keys add mocha-4 batch --bech32=celestia`

*Make sure that this new account has a sufficient balance.*

### Step 2.	Run the Update batch info command. 
`opinitd tx update-batch-info CELESTIA celestia...`

The proposer sends a transaction that includes an update batch info message with the new `ChainType` and `SubmitterAddress`. The bot will `shut down` when it encounters the `MsgUpdateBatchInfo`.

### Step 3. Update the DA node config in the executor config.
```json
{
...
  "da_node": {
    "chain_id": "mocha-4",
    "bech32_prefix": "celestia",
    "rpc_address": "",
    "gas_price": "",
    "gas_adjustment": 1.5,
    "tx_timeout": 60
  }
}
```

### Step 4. Start the OPInit bot.