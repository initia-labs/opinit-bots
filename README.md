# OPinit Bots

This repository contains the Go implementation of OPinit bots.

## Components

- [Executor](./executor)
- [Challenger](./challenger)

## How to Use

### Prerequisites

Before running OPinit bots, make sure you have the following prerequisites installed:

- Go 1.22.2+

To ensure compatibility with the node version, check the following versions:

| L1 Node | MiniMove | MiniWasm | MiniEVM | 
| ------- | -------- | -------- | ------- | 
| v0.4.7  | v0.4.1   | v0.4.1   |    -    | 

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

To reset bot's specific node height info to 0 , use the following command:

```bash
opinitd reset-height [bot-name] [node-type]

Executor node types: 
- host
- child
- batch

Challenger node types: 
- host
- child
```
