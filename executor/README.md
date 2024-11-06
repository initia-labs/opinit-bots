# Executor

The Executor is responsible for

1. relaying L1 token deposit transactions to L2 and periodically submitting L2 output roots to L1.
2. relaying L1 oracle data to L2.
3. submitting L2 block data to DA.
4. providing withdrawal proofs for user's token withdrawals.

## Config

To configure the Executor, fill in the values in the `~/.opinit/executor.json` file.

```json
{
  // Version is the version used to build output root.
  // Please refer to `spec_version.json` for the correct version for each network.
  "version": 1,
  // ListenAddress is the address to listen for incoming requests.
  "listen_address": "localhost:3000",
  "l1_node": {
    "chain_id": "testnet-l1-1",
    "bech32_prefix": "init",
    "rpc_address": "tcp://localhost:26657",
    "gas_price": "0.15uinit",
    "gas_adjustment": 1.5,
    "tx_timeout": 60
  },
  "l2_node": {
    "chain_id": "testnet-l2-1",
    "bech32_prefix": "init",
    "rpc_address": "tcp://localhost:27657",
    "gas_price": "",
    "gas_adjustment": 1.5,
    "tx_timeout": 60
  },
  "da_node": {
    "chain_id": "testnet-l1-1",
    "bech32_prefix": "init",
    "rpc_address": "tcp://localhost:26657",
    "gas_price": "0.15uinit",
    "gas_adjustment": 1.5,
    "tx_timeout": 60
  },
  // BridgeExecutor is the key name in the keyring for the bridge executor,
  // which is used to relay initiate token bridge transaction from l1 to l2.
  //
  // If you don't want to use the bridge executor feature, you can leave it empty.
  "bridge_executor": "",

	// DisableOutputSubmitter is the flag to disable the output submitter.
	// If it is true, the output submitter will not be started.
	"disable_output_submitter": false,

	// DisableBatchSubmitter is the flag to disable the batch submitter.
	// If it is true, the batch submitter will not be started.
	"disable_batch_submitter": false,

  // MaxChunks is the maximum number of chunks in a batch.
  "max_chunks": 5000,
  // MaxChunkSize is the maximum size of a chunk in a batch.
  "max_chunk_size": 300000,
  // MaxSubmissionTime is the maximum time to submit a batch.
  "max_submission_time": 3600,
  // DisableAutoSetL1Height is the flag to disable the automatic setting of the l1 height.
  // If it is false, it will finds the optimal height and sets l1_start_height automatically
  // from l2 start height and l1_start_height is ignored.
  // It can be useful when you don't want to use TxSearch.
  "disable_auto_set_l1_height": false,
  // L1StartHeight is the height to start the l1 node.
  "l1_start_height": 0,
  // L2StartHeight is the height to start the l2 node. If it is 0, it will start from the latest height.
  // If the latest height stored in the db is not 0, this config is ignored.
  // L2 starts from the last submitted output l2 block number + 1 before L2StartHeight.
  // L1 starts from the block number of the output tx + 1
  "l2_start_height": 0,
  // StartBatchHeight is the height to start the batch. If it is 0, it will start from the latest height.
  // If the latest height stored in the db is not 0, this config is ignored.
  "batch_start_height": 0
}
```

### Start height config examples

If the latest height stored in the db is not 0, start height config is ignored.

```shell
Output tx 1 
- L1BlockNumber: 10
- L2BlockNumber: 100

Output tx 2
- L1BlockNumber: 20
- L2BlockNumber: 200

InitializeTokenDeposit tx 1
- Height: 5
- L1Sequence: 1

InitializeTokenDeposit tx 2
- Height: 15
- L1Sequence: 2

FinalizedTokenDeposit tx 1
- L1Sequence: 1

FinalizedTokenDeposit tx 2
- L1Sequence: 2
```

#### Config 1

```json
{
  l2_start_height: 150, 
  batch_start_height: 0
}
```

When Child's last l1 Sequence is `2`,

- L1 starts from the height 10 + 1 = 11
- L2 starts from the height 100 + 1 = 101
- Batch starts from the height 1

#### Config 2

```json
{
  l2_start_height: 150, 
  batch_start_height: 150
}
```

When Child's last l1 Sequence is `2`,

- L1 starts from the height 10 + 1 = 11
- L2 starts from the height 100 + 1 = 101
- Batch starts from the height 150

#### Config 3

```json
{
  l2_start_height: 150, 
  batch_start_height: 150
}
```

When Child's last l1 Sequence is `1`,

- L1 starts from the height 5 + 1 = 6
- L2 starts from the height 100 + 1 = 101
- Batch starts from the height 150

## Handler rules for the components of the Executor

For registered events or tx handlers, work processed in a block is atomically saved as ProcessedMsg. Therefore, if ProcessedMsgs or Txs cannot be processed due to an interrupt or error, it is guaranteed to be read from the DB and processed.

## Deposit

When the `initiate_token_deposit` event is detected in l1, `bridge_executor` submits a tx containing the `MsgFinalizeTokenDeposit` msg to l2.

## Withdrawal

`Child` always has a merkle tree. A working tree is stored for each block, and if the working tree of the previous block does not exist, a `panic` occurs. When it detects an `initiate_token_withdrawal` event, it adds it as a leaf node to the current working tree. The leaf index of the current working tree corresponding to the requested withdrawal is `l2 sequence of the withdrawal - start index of the working tree`. To add a leaf node, it calculates the leaf node with the withdrawal hash such as the [opinit spec](https://github.com/initia-labs/OPinit/blob/v0.4.3/x/ophost/types/output.go#L30) and stores the withdrawal info corresponding to the l2 sequence. For rules on creating trees, refer [this](../merkle).

```go
func GenerateWithdrawalHash(bridgeId uint64, l2Sequence uint64, sender string, receiver string, denom string, amount uint64) [32]byte {
 var withdrawalHash [32]byte
 seed := []byte{}
 seed = binary.BigEndian.AppendUint64(seed, bridgeId)
 seed = binary.BigEndian.AppendUint64(seed, l2Sequence)

 // variable length
 senderDigest := sha3.Sum256([]byte(sender))
 seed = append(seed, senderDigest[:]...) // put utf8 encoded address
 // variable length
 receiverDigest := sha3.Sum256([]byte(receiver))
 seed = append(seed, receiverDigest[:]...) // put utf8 encoded address
 // variable length
 denomDigest := sha3.Sum256([]byte(denom))
 seed = append(seed, denomDigest[:]...)
 seed = binary.BigEndian.AppendUint64(seed, amount)

 // double hash the leaf node
 withdrawalHash = sha3.Sum256(seed)
 withdrawalHash = sha3.Sum256(withdrawalHash[:])

 return withdrawalHash
}
```

When `2/3` of the submission interval registered in the chain has passed since the previous submission time, the child finalizes the current working tree and submits the output root created with the root of the tree as a storage root. Currently, version `0` is used.

```go
func GenerateOutputRoot(version byte, storageRoot []byte, latestBlockHash []byte) [32]byte {
 seed := make([]byte, 1+32+32)
 seed[0] = version
 copy(seed[1:], storageRoot[:32])
 copy(seed[1+32:], latestBlockHash[:32])
 return sha3.Sum256(seed)
}
```

When a tree is finalized, `Child` stores the leaf nodes and internal nodes of the tree to provide withdrawal proofs. When a user queries for a withdrawal of a sequence, the following response is returned:

```go
type QueryWithdrawalResponse struct {
 // fields required to withdraw funds
 BridgeId         uint64   `json:"bridge_id"`
 OutputIndex      uint64   `json:"output_index"`
 WithdrawalProofs [][]byte `json:"withdrawal_proofs"`
 Sender           string   `json:"sender"`
 Sequence         uint64   `json:"sequence"`
 Amount           string   `json:"amount"`
 Version          []byte   `json:"version"`
 StorageRoot      []byte   `json:"storage_root"`
 LatestBlockHash  []byte   `json:"latest_block_hash"`

 // extra info
 BlockNumber    int64 `json:"block_number"`
 Receiver       string `json:"receiver"`
 WithdrawalHash []byte `json:"withdrawal_hash"`
}
```

This data contains all the data needed to finalize withdrawal.

## Oracle

Initia uses [connect@v2](https://github.com/skip-mev/connect) to bring oracle data into the chain, which is stored in the 0th tx of each block. The bridge executor submits a `MsgUpdateOracle` containing the 0th Tx of l1 block to l2 when a block in l1 is created. Since oracle data always needs to be the latest, old oracles are discarded or ignored. To relay oracle, `oracle_enabled` must be set to true in bridge config.

## Batch

`Batch` queries the batch info stored in the chain and submit the batch according to the account and chain ID. The user must provide the appropriate `RPC address`, `bech32-prefix` and `gas-price` via config. Also, the account in the batch info must be registered in the keyring. Each block's raw bytes is compressed with `gzip`. The collected block data is divided into max chunk size of config. When the `2/3` of the submission interval registered in the chain has passed since the previous submission time, it submits the batch data header first and batch data chunks to DA by adding last raw commit bytes with headers. The batch header contains the start, end l2 block height and the checksums of each chunk that this data contains.

```go
// BatchDataHeader is the header of a batch
type BatchDataHeader struct {
 Start     uint64
 End       uint64
 Checksums [][]byte
}

func MarshalBatchDataHeader(
 start uint64,
 end uint64,
 checksums [][]byte,
) []byte {
 data := make([]byte, 1)
 data[0] = byte(BatchDataTypeHeader)
 data = binary.BigEndian.AppendUint64(data, start)
 data = binary.BigEndian.AppendUint64(data, end)
 data = binary.BigEndian.AppendUint64(data, uint64(len(checksums)))
 for _, checksum := range checksums {
  data = append(data, checksum...)
 }
 return data
}

// BatchDataChunk is the chunk of a batch
type BatchDataChunk struct {
 Start     uint64
 End       uint64
 Index     uint64
 Length    uint64
 ChunkData []byte
}

func MarshalBatchDataChunk(
 start uint64,
 end uint64,
 index uint64,
 length uint64,
 chunkData []byte,
) []byte {
 data := make([]byte, 1)
 data[0] = byte(BatchDataTypeChunk)
 data = binary.BigEndian.AppendUint64(data, start)
 data = binary.BigEndian.AppendUint64(data, end)
 data = binary.BigEndian.AppendUint64(data, index)
 data = binary.BigEndian.AppendUint64(data, length)
 data = append(data, chunkData...)
 return data
}
```

### Note

- If a l2 block contains `MsgUpdateOracle`, only the data field is submitted empty to reduce block bytes since the oracle data is already stored in l1.
- Batch data is stored in a `batch` file in the home directory until it is submitted, so be careful **not to change the file.**

### Update batch info

If the batch info registered in the chain is changed to change the account or DA chain for the batch, `Host` catches the `update_batch_info` event and send it to `Batch`. The batch will empty the temporal batch file and turn off the bot to resubmit from the last finalized output block number. Users must update the config file with updated information before starting the bot.

```go
{
 RPCAddress string `json:"rpc_address"`
 GasPrice string `json:"gas_price"`
 GasAdjustment string `json:"gas_adjustment"`
 ChainID string `json:"chain_id"`
 Bech32Prefix string `json:"bech32_prefix"`
}
```

## Sync from the beginning

If for some reason you need to re-sync from the beginning, the bot will query the outputs and deposits submitted to the chain and not resubmit them. However, the tree must always be saved, as it must provide withdrawal proofs.

## Query

### Status

```bash
curl localhost:3000/status
```

```json
{
  "bridge_id": 1,
  "host": {
    "node": {
      "last_block_height": 0,
      "broadcaster": {
        "pending_txs": 0,
        "sequence": 0
      }
    },
    "last_proposed_output_index": 0,
    "last_proposed_output_l2_block_number": 0
  },
  "child": {
    "node": {
      "last_block_height": 0,
      "broadcaster": {
        "pending_txs": 0,
        "sequence": 0
      }
    },
    "last_updated_oracle_height": 0,
    "last_finalized_deposit_l1_block_height": 0,
    "last_finalized_deposit_l1_sequence": 0,
    "last_withdrawal_l2_sequence": 0,
    "working_tree_index": 0,
    "finalizing_block_height": 0,
    "last_output_submission_time": "",
    "next_output_submission_time": ""
  },
  "batch": {
    "node": {
      "last_block_height": 0,
    },
    "batch_info": {
      "submitter": "",
      "chain_type": ""
    },
    "current_batch_file_size": 0,
    "batch_start_block_number": 0,
    "batch_end_block_number": 0,
    "last_batch_submission_time": ""
  },
  "da": {
    "broadcaster": {
      "pending_txs": 0,
      "sequence": 0
    }
  }
}
```

### Withdrawals

```bash
curl localhost:3000/withdrawal/{sequence} | jq . > ./withdrawal-info.json
initiad tx ophost finalize-token-withdrawal ./withdrawal-info.json --gas= --gas-prices= --chain-id= --from=
```

```go
type QueryWithdrawalResponse struct {
 // fields required to withdraw funds
 BridgeId         uint64   `json:"bridge_id"`
 OutputIndex      uint64   `json:"output_index"`
 WithdrawalProofs [][]byte `json:"withdrawal_proofs"`
 Sender           string   `json:"sender"`
 Sequence         uint64   `json:"sequence"`
 Amount           string   `json:"amount"`
 Version          []byte   `json:"version"`
 StorageRoot      []byte   `json:"storage_root"`
 LatestBlockHash  []byte   `json:"latest_block_hash"`

 // extra info
 BlockNumber    int64 `json:"block_number"`
 Receiver       string `json:"receiver"`
 WithdrawalHash []byte `json:"withdrawal_hash"`
}
```
