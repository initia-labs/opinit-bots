# Challenger

The Challenger is responsible for

1. verifying that the `MsgInitiateTokenDeposit` event is properly relayed to `MsgFinalizeTokenDeposit`.
2. checking whether `MsgInitiateTokenDeposit` was relayed on time.
3. verifying that the `Oracle` data is properly relayed to `MsgUpdateOracle`.
4. checking whether `Oracle` was relayed on time.
5. verifying that the `OutputRoot` submitted with `MsgProposeOutput` is correct.
6. checking whether next `MsgProposeOutput` was submitted on time.

## Config

To configure the Challenger, fill in the values in the `~/.opinit/challenger.json` file.

```json
{
  // Version is the version used to build output root.
  // Please refer to `spec_version.json` for the correct version for each network.
  "version": 1,
  // Server is the configuration for the server.
  "server": {
    "address":      "localhost:3000",
    "allow_origins": "*",
    "allow_headers": "Origin, Content-Type, Accept",
    "allow_methods": "GET",
  },
  "l1_node": {
    "chain_id": "testnet-l1-1",
    "bech32_prefix": "init",
    "rpc_address": "tcp://localhost:26657",
  },
  "l2_node": {
    "chain_id": "testnet-l2-1",
    "bech32_prefix": "init",
    "rpc_address": "tcp://localhost:27657",
  },
  // DisableAutoSetL1Height is the flag to disable the automatic setting of the l1 height.
  // If it is false, it will finds the optimal height and sets l1_start_height automatically
  // from l2 start height and l1_start_height is ignored.
  // It can be useful when you don't want to use TxSearch.
  "disable_auto_set_l1_height": false,
  // L1StartHeight is the height to start the l1 node.
  "l1_start_height": 1,
  // L2StartHeight is the height to start the l2 node. 
  // If the latest height stored in the db is not 0, this config is ignored.
  // L2 starts from the last submitted output l2 block number + 1 before L2StartHeight.
  // L1 starts from the block number of the output tx + 1
  "l2_start_height": 1,
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
}
```

When Child's last l1 Sequence is `2`,

- L1 starts from the height 10 + 1 = 11
- L2 starts from the height 100 + 1 = 101

## Handler rules for the components of the Challenger

For registered events or tx handlers, work processed in a block is atomically saved as `pending events`. Therefore, if `pending events` with the `ChallengeEvent` interface cannot be processed due to an interrupt or error, it is guaranteed to be read from the DB and processed. When an event matching the pending event comes in and is processed, or when the block time exceeds the event's timeout, a `Challenge` is created and stored in the DB.

### The challenger can check the generated `Challenges` and decide what action to take

## Deposit

When the `initiate_token_deposit` event is detected in l1, saves it as a `Deposit` challenge event and check if it is the same as the `MsgFinalizeTokenDeposit` for the same sequence.

```go
// Deposit is the challenge event for the deposit
type Deposit struct {
 EventType     string    `json:"event_type"`
 Sequence      uint64    `json:"sequence"`
 L1BlockHeight int64    `json:"l1_block_height"`
 From          string    `json:"from"`
 To            string    `json:"to"`
 L1Denom       string    `json:"l1_denom"`
 Amount        string    `json:"amount"`
 Time          time.Time `json:"time"`
 Timeout       bool      `json:"timeout"`
}
```

## Output

When the `propose_output` event is detected in l1, saves it as a `Output` challenge event, replays up to l2 block number and check if `OutputRoot` is the same as submitted.

```go
// Output is the challenge event for the output
type Output struct {
 EventType     string    `json:"event_type"`
 L2BlockNumber int64    `json:"l2_block_number"`
 OutputIndex   uint64    `json:"output_index"`
 OutputRoot    []byte    `json:"output_root"`
 Time          time.Time `json:"time"`
 Timeout       bool      `json:"timeout"`
}
```

## Oracle

If `oracle_enable` is turned on in bridge config, saves bytes of the 0th Tx as a `Oracle` challenge event and check if it is the same data in the `MsgUpdateOracle` for the l1 height.

## Batch

Batch data is not verified by the challenger bot.

### TODO

- Challenger runs a L2 node it in rollup sync challenge mode in CometBFT to check whether the submitted batch is replayed properly.

## Query

### Status

```bash
curl localhost:3001/status
```

```json
{
  "bridge_id": 0,
  "host": {
    "node": {
      "last_block_height": 0
    },
    "last_output_index": 0,
    "last_output_time": "",
    "num_pending_events": {}
  },
  "child": {
    "node": {
      "last_block_height": 0
    },
    "last_updated_oracle_height": 0,
    "last_finalized_deposit_l1_block_height": 0,
    "last_finalized_deposit_l1_sequence": 0,
    "last_withdrawal_l2_sequence": 0,
    "working_tree_index": 0,
    "finalizing_block_height": 0,
    "last_output_submission_time": "",
    "next_output_submission_time": "",
    "num_pending_events": {}
  },
  "latest_challenges": []
}
```

### Challenges

```bash
curl localhost:3001/challenges
```

default options
- `limit`: 10
- `next`: ""
- `order`: desc

```go
type QueryChallengesResponse struct {
  Challenges []Challenge `json:"challenges"`
  Next       *string     `json:"next,omitempty"`
}
```

If `next` exists, you can continue querying by inserting it as the `next`.

```json
[
  {
    "event_type": "Oracle",
    "id": {
      "type": 2,
      "id": 136
    },
    "log": "event timeout: Oracle{L1Height: 136, Data: nUYSP4e3jTUBk33jFSYuWW28U+uTO+g3IO5/iZfbDTo, Time: 2024-09-11 06:01:21.257012 +0000 UTC}",
    "timestamp": "2024-09-11T06:05:21.284479Z"
  },
]
```

### Pending events

```bash
curl localhost:3001/pending_events/host
```

```json
[
  {
    "event_type": "Output",
    "l2_block_number": 2394,
    "output_index": 7,
    "output_root": "Lx+k0GuHi6jPcO5yA6dUwI/l0h4b25awy973F6CirYs=",
    "time": "2024-09-11T06:28:54.186941Z",
    "timeout": false
  }
]
```

```bash
curl localhost:3001/pending_events/child
```

```json
[
  {
    "event_type": "Oracle",
    "l1_height": 1025,
    "data": "49Y7iryZGsQYzpao+rxR2Vfaz6owp3s5vlPnkboXwr0=",
    "time": "2024-09-11T06:31:28.657245Z",
    "timeout": false
  }
]
```
