/*
Package cmd includes relayer commands
Copyright © 2020 Jack Zampolin jack.zampolin@gmail.com

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/initia-labs/opinit-bots-go/executor/celestia"
	"github.com/initia-labs/opinit-bots-go/executor/child"
	"github.com/initia-labs/opinit-bots-go/executor/host"
	"github.com/initia-labs/opinit-bots-go/node"
	"github.com/spf13/cobra"
)

const (
	flagRestore     = "restore"
	flagMnemonicSrc = "source"
)

// keysCmd represents the keys command
func keysCmd(ctx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "keys",
		Aliases: []string{"k"},
		Short:   "Manage keys held by the opbot",
	}

	cmd.AddCommand(
		keysAddCmd(ctx),
		keysListCmd(ctx),
		keysShowCmd(ctx),
		keysShowByAddressCmd(ctx),
		keysDeleteCmd(ctx),
	)

	return cmd
}

// keysAddCmd represents the `keys add` command
func keysAddCmd(ctx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "add [chain-id] [key-name]",
		Aliases: []string{"a"},
		Short:   "Adds a key to the keychain associated with a particular chain",
		Args:    cobra.ExactArgs(2),
		Example: strings.TrimSpace(`
$ keys add localnet key1
$ keys add l2 key2
$ keys add l2 key2 --restore mnemonic.txt`),
		RunE: func(cmd *cobra.Command, args []string) error {
			chainId := args[0]
			keyName := args[1]

			cdc, prefix, err := GetCodec(chainId)
			if err != nil {
				return err
			}

			keyBase, err := node.GetKeyBase(chainId, ctx.homePath, cdc, cmd.InOrStdin())
			if err != nil {
				return err
			}

			account, err := keyBase.Key(keyName)
			if err == nil && account.Name == keyName {
				return fmt.Errorf("key with name %s already exists", keyName)
			}

			mnemonic := ""
			mnemonicSrc, err := cmd.Flags().GetString(flagRestore)
			if err == nil && mnemonicSrc != "" {
				file, err := os.Open(mnemonicSrc)
				if err != nil {
					return err
				}
				defer file.Close()

				bz, err := io.ReadAll(file)
				if err != nil {
					return err
				}
				mnemonic = strings.TrimSpace(string(bz))
			} else {
				mnemonic, err = node.CreateMnemonic()
				if err != nil {
					return err
				}
			}

			account, err = keyBase.NewAccount(keyName, mnemonic, "", hd.CreateHDPath(sdk.CoinType, 0, 0).String(), hd.Secp256k1)
			if err != nil {
				return err
			}

			addr, err := account.GetAddress()
			if err != nil {
				return err
			}

			addrString, err := node.EncodeBech32AccAddr(addr, prefix)
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n%s\n", account.Name, addrString, mnemonic)
			return nil
		},
	}
	cmd.Flags().String(flagRestore, "", "restores from mnemonic file source")
	return cmd
}

// keysListCmd represents the `keys list` command
func keysListCmd(ctx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list [chain-id]",
		Aliases: []string{"l"},
		Short:   "Lists keys from the keychain associated with a particular chain",
		Args:    cobra.ExactArgs(1),
		Example: strings.TrimSpace(`
$ keys list localnet
$ k l l2`),
		RunE: func(cmd *cobra.Command, args []string) error {
			chainId := args[0]

			cdc, prefix, err := GetCodec(chainId)
			if err != nil {
				return err
			}

			keyBase, err := node.GetKeyBase(chainId, ctx.homePath, cdc, cmd.InOrStdin())
			if err != nil {
				return err
			}

			info, err := keyBase.List()
			if err != nil {
				return err
			}

			if len(info) == 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "no keys found for %s\n", chainId)
			}

			for _, account := range info {
				addr, err := account.GetAddress()
				if err != nil {
					return err
				}

				addrString, err := node.EncodeBech32AccAddr(addr, prefix)
				if err != nil {
					return err
				}

				fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", account.Name, addrString)
			}

			return nil
		},
	}

	return cmd
}

// keysShowCmd represents the `keys show` command
func keysShowCmd(ctx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "show [chain-id] [key-name]",
		Aliases: []string{"s"},
		Short:   "Shows the key from the keychain associated with a particular chain",
		Args:    cobra.ExactArgs(2),
		Example: strings.TrimSpace(`
$ keys show localnet key1
$ k s l2 key2`),
		RunE: func(cmd *cobra.Command, args []string) error {
			chainId := args[0]
			keyName := args[1]

			cdc, prefix, err := GetCodec(chainId)
			if err != nil {
				return err
			}

			keyBase, err := node.GetKeyBase(chainId, ctx.homePath, cdc, cmd.InOrStdin())
			if err != nil {
				return err
			}

			account, err := keyBase.Key(keyName)
			if err == nil && account.Name == keyName {
				addr, err := account.GetAddress()
				if err != nil {
					return err
				}

				addrString, err := node.EncodeBech32AccAddr(addr, prefix)
				if err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", account.Name, addrString)
				return nil
			}
			return fmt.Errorf("key with name %s does not exist", keyName)
		},
	}

	return cmd
}

// keysShowByAddressCmd represents the `keys show-by-address` command
func keysShowByAddressCmd(ctx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "show-by-addr [chain-id] [key-address]",
		Aliases: []string{"sa"},
		Short:   "Shows the key by address from the keychain associated with a particular chain",
		Args:    cobra.ExactArgs(2),
		Example: strings.TrimSpace(`
$ keys show-by-addr localnet key1
$ k sa l2 key2`),
		RunE: func(cmd *cobra.Command, args []string) error {
			chainId := args[0]
			keyAddr := args[1]

			cdc, prefix, err := GetCodec(chainId)
			if err != nil {
				return err
			}

			keyBase, err := node.GetKeyBase(chainId, ctx.homePath, cdc, cmd.InOrStdin())
			if err != nil {
				return err
			}

			addr, err := node.DecodeBech32AccAddr(keyAddr, prefix)
			if err != nil {
				return err
			}

			account, err := keyBase.KeyByAddress(addr)
			if err != nil {
				return fmt.Errorf("key with address %s does not exist", keyAddr)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", account.Name, keyAddr)
			return nil
		},
	}

	return cmd
}

// keysDeleteCmd represents the `keys delete` command
func keysDeleteCmd(ctx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete [chain-id] [key-name]",
		Aliases: []string{"d"},
		Short:   "Deletes the key from the keychain associated with a particular chain",
		Args:    cobra.ExactArgs(2),
		Example: strings.TrimSpace(`
$ keys delete localnet key1
$ k d l2 key2`),
		RunE: func(cmd *cobra.Command, args []string) error {
			chainId := args[0]
			keyName := args[1]

			cdc, prefix, err := GetCodec(chainId)
			if err != nil {
				return err
			}

			keyBase, err := node.GetKeyBase(chainId, ctx.homePath, cdc, cmd.InOrStdin())
			if err != nil {
				return err
			}

			account, err := keyBase.Key(keyName)
			if err == nil && account.Name == keyName {
				addr, err := account.GetAddress()
				if err != nil {
					return err
				}

				addrString, err := node.EncodeBech32AccAddr(addr, prefix)
				if err != nil {
					return err
				}

				err = keyBase.Delete(keyName)
				if err != nil {
					return err
				}

				fmt.Fprintf(cmd.OutOrStdout(), "%s: %s deleted\n", account.Name, addrString)
				return nil
			}
			return fmt.Errorf("key with name %s does not exist", keyName)
		},
	}
	return cmd
}

func GetCodec(chainId string) (codec.Codec, string, error) {
	switch chainId {
	case "initiation-1":
		cdc, _, prefix := host.GetCodec()
		return cdc, prefix, nil

	case "celestia", "arabica-11":
		cdc, _, prefix := celestia.GetCodec()
		return cdc, prefix, nil

	// for test
	case "localnet":
		cdc, _, prefix := host.GetCodec()
		return cdc, prefix, nil

	default:
		cdc, _, prefix, err := child.GetCodec(chainId)
		return cdc, prefix, err
	}

	// return nil, "", fmt.Errorf("unsupported chain id: %s", chainId)
}
