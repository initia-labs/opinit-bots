# OPinit Bots Go

This repository contains the Go implementation of OPinit bots.

## Components

- [Executor](./executor)
- Batch Submitter
- Challenger

## How to Use

### Prerequisites

Before running OPinit bots, make sure you have the following prerequisites installed:

- Go 1.22.2+
- Node RPC (L1 and L2)

To ensure compatibility with the node version, check the following versions:

| L1 Node | MiniMove | MiniWasm | MiniEVM | OPinit-bots |
| ------- | -------- | -------- | ------- | ----------- |
| v0.4.0  | v0.4.0   | v0.4.0   | v0.4.0  | v0.1.0      |

### Build and Configure

To build and configure the bots, follow these steps:

```bash
make install

# Default config path is ~/.opinit/[bot-name].json
# - Customize home dir with --home ~/.opinit-custom-path
# - Customize config name with --config [bot-custom-name].json
#
# Supported bot names
# - executor
opinitd init [bot-name]
```

### Start Bot Program

To start the bot program, use the following command:

```bash
opinitd start [bot-name]
```
