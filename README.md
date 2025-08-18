# OPinit Bots

This repository contains the Go implementation of OPinit bots.

## Components

- [Executor](./executor)
- [Challenger](./challenger)

## How to Use

### Prerequisites

Before running OPinit bots, make sure you have the following prerequisites installed:

- Go 1.23.6+

To ensure compatibility with the node version, check the following versions:

| L1 Node | MiniMove | MiniWasm | MiniEVM | Celestia-appd |
| ------- | -------- | -------- | ------- | ------------- |
| v1.0.0+ | v1.0.0+  | v1.0.0+  | v1.0.0+ | v3.3.2+       |

### Build and Configure

To build and configure the bots, follow these steps:

```bash
make install
opinitd init [bot-name]
```

Default config path is `~/.opinit/[bot-name].json`

- Customize home dir with `--home ~/.opinit-custom-path`
- Customize config name with `--config [bot-custom-name].json`

Supported bot names

- `executor`
- `challenger`

### Register keys

```bash
opinitd keys add [chain-id] [key-name]
### with mnemonic file
opinitd keys add [chain-id] [key-name] --recover --source [mnemonic-file-path]
### with mnemonic input
opinitd keys add [chain-id] [key-name] --recover
### with bech32 prefix
opinitd keys add [chain-id] [key-name] --bech32=celestia
```

### Start Bot

To start the bot, use the following command:

```bash
opinitd start [bot-name]
```

Options

- `--log-level`: log level can be set. Default log level is `info`.
- `--polling-interval`: polling interval can be set. Default polling interval is `100ms`.
- `--config`: config file name can be set. Default config file name is `[bot-name].json`.
- `--home`: home dir can be set. Default home dir is `~/.opinit`.
  
### Reset Bot DB

To reset the bot database, use the following command:

```bash
opinitd reset-db [bot-name]
```

### Reset Node Heights

To reset bot's all height info to 0, use the following command:

```bash
opinitd reset-heights [bot-name]
```

### Reset Node Height

To reset bot's specific node height info to 0, use the following command:

```bash
opinitd reset-height [bot-name] [node-type]

Executor node types: 
- host
- child
- batch
- da

Challenger node types: 
- host
- child
```

## RPC Pool Configuration

To enhance reliability, the bots support multiple RPC endpoints for L1, L2, and DA nodes. This allows for automatic fallback and retries if an endpoint becomes unavailable.

### Endpoint Configuration

In your configuration file (`~/.opinit/[bot-name].json`), you can specify a list of RPC addresses for each node:

```json
{
  "l1_node": {
    "chain_id": "testnet-l1-1",
    "bech32_prefix": "init",
    "rpc_addresses": [
      "tcp://doi-rpc:26657",
      "tcp://another-l1-rpc:26657"
    ]
  },
  "l2_node": {
    "chain_id": "testnet-l2-1",
    "bech32_prefix": "init",
    "rpc_addresses": [
      "tcp://moro-rpc:27657",
      "tcp://another-l2-rpc:27657"
    ]
  },
  "da_node": {
    "chain_id": "testnet-l1-1",
    "bech32_prefix": "init",
    "rpc_addresses": [
      "tcp://rene-rpc:26657",
      "tcp://another-da-rpc:26657"
    ]
  }
}
```

The bot will try the endpoints in the order they are listed. If a request to the first endpoint fails, it will automatically fall back to the next one in the list.

### Timeout Configuration

You can configure the timeout for each RPC request using the `RPC_TIMEOUT_SECONDS` environment variable. The value is in seconds. If the variable is not set, the default timeout is 5 seconds.

```bash
export RPC_TIMEOUT_SECONDS=10
opinitd start [bot-name]
```
