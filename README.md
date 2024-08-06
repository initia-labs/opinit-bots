# OPinit Bots Go

This repository contains the Go implementation of OPinit bots.

## Components

- [Executor](./executor)

## How to Use

### Prerequisites

Before running OPinit bots, make sure you have the following prerequisites installed:

- Go 1.22.2+

To ensure compatibility with the node version, check the following versions:

| L1 Node | MiniMove | MiniWasm | MiniEVM | 
| ------- | -------- | -------- | ------- | 
| v0.4.1  | v0.4.1   | v0.4.1   | v0.4.1  | 

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
log level can be set by using `--log-level` flag. Default log level is `info`.

### Reset Bot DB

To reset the bot database, use the following command:

```bash
opinitd reset-db [bot-name]
```

### Query withdrawals 
```bash
curl localhost:3000/withdrawal/{sequence} | jq . > ./withdrawal-info.json
initiad tx ophost finalize-token-withdrawal ./withdrawal-info.json --gas= --gas-prices= --chain-id= --from=
```