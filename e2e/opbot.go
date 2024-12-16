package e2e

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/client"
	"go.uber.org/zap"

	"github.com/initia-labs/opinit-bots/executor"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
)

const (
	DefaultUIDGID = "1000:1000"
)

type OPBot struct {
	*DockerOPBot
}

func NewOPBot(log *zap.Logger, botName string, testName string, cli *client.Client, networkID string) *OPBot {
	c := &commander{log: log}

	op, err := NewDockerOPBot(context.Background(), log, botName, testName, cli, networkID, c, true)
	if err != nil {
		panic(err) // TODO: return
	}

	c.extraStartFlags = op.GetExtraStartupFlags()

	r := &OPBot{
		DockerOPBot: op,
	}
	return r
}

func (op *OPBot) WaitForSync(ctx context.Context) error {
	timer := time.NewTicker(5 * time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:

		}

		syncing, err := op.QuerySyncing()
		if err != nil {
			continue
		}
		if !syncing {
			break
		}
	}
	return nil
}

func query(address string, params map[string]string) ([]byte, error) {
	u, err := url.Parse(address)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (op *OPBot) QuerySyncing() (bool, error) {
	address := op.DockerOPBot.queryServerUrl + "/syncing"

	data, err := query(address, nil)
	if err != nil {
		return false, err
	}

	return strconv.ParseBool(string(data))
}

func (op *OPBot) QueryExecutorStatus() (executor.Status, error) {
	address := op.DockerOPBot.queryServerUrl + "/status"

	data, err := query(address, nil)
	if err != nil {
		return executor.Status{}, err
	}

	var status executor.Status
	err = json.Unmarshal(data, &status)
	if err != nil {
		return executor.Status{}, err
	}
	return status, nil
}

const (
	DefaultContainerImage   = "ghcr.io/initia-labs/opinitd"
	DefaultContainerVersion = "v0.1.11"
)

type commander struct {
	log             *zap.Logger
	extraStartFlags []string
}

func (commander) Name() string {
	return "opinit-bot"
}

func (commander) DockerUser() string {
	return DefaultUIDGID
}

func (commander) AddKey(chainID, keyName, bech32Prefix, homeDir string) []string {
	return []string{
		"opinitd", "keys", "add", chainID, keyName,
		"--bech32", bech32Prefix,
		"--home", homeDir,
		"--output", "json",
	}
}

func (c commander) RestoreKey(chainID, keyName, bech32Prefix, mnemonic, homeDir string) []string {
	cmd := c.AddKey(chainID, keyName, bech32Prefix, homeDir)
	cmd = append(cmd, "--recover")
	cmd = append(cmd, "--mnemonic")
	cmd = append(cmd, mnemonic)
	return cmd
}

func (c commander) Start(botName string, homeDir string) []string {
	return []string{
		"opinitd", "start", botName,
		"--log-level", "debug",
		"--home", homeDir,
	}
}

func (commander) DefaultContainerImage() string {
	return DefaultContainerImage
}

func (commander) DefaultContainerVersion() string {
	return DefaultContainerVersion
}

func (commander) ParseAddKeyOutput(stdout, stderr string) (ibc.Wallet, error) {
	var wallet WalletModel
	err := json.Unmarshal([]byte(stdout), &wallet)

	for keyName, elem := range wallet {
		opWallet := NewWallet(keyName, elem.Address, elem.Mnemonic)
		return opWallet, err
	}
	return nil, errors.New("failed to parse wallet")
}

func (commander) ParseRestoreKeyOutput(stdout, stderr string) string {
	return strings.Replace(stdout, "\n", "", 1)
}

func (commander) Init(botName string, homeDir string) []string {
	return []string{
		"opinitd", "init", botName,
		"--home", homeDir,
	}
}

func (c commander) CreateWallet(keyName, address, mnemonic string) ibc.Wallet {
	return NewWallet(keyName, address, mnemonic)
}

var _ ibc.Wallet = &OPWallet{}

type WalletModel map[string]struct {
	Mnemonic string `json:"mnemonic"`
	Address  string `json:"address"`
}

type OPWallet struct {
	mnemonic string
	address  string
	keyName  string
}

func NewWallet(keyname string, address string, mnemonic string) *OPWallet {
	return &OPWallet{
		mnemonic: mnemonic,
		address:  address,
		keyName:  keyname,
	}
}

func (op *OPWallet) KeyName() string {
	return op.keyName
}

func (op *OPWallet) FormattedAddress() string {
	return op.address
}

func (op *OPWallet) Mnemonic() string {
	return op.mnemonic
}

// Get Address.
func (op *OPWallet) Address() []byte {
	return []byte(op.address)
}
