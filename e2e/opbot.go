package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/client"
	"go.uber.org/zap"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/initia-labs/opinit-bots/executor"
	executortypes "github.com/initia-labs/opinit-bots/executor/types"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"

	"github.com/pkg/errors"
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
		if err != nil || syncing {
			op.log.Error("query syncing result", zap.Bool("syncing", syncing), zap.Error(err))
			continue
		}
		break
	}
	return nil
}

func (op *OPBot) query(address string, params map[string]string) (data []byte, err error) {
	u, err := url.Parse(address)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	defer func() {
		op.log.Info("opbot query", zap.String("url", u.String()), zap.String("data", string(data)), zap.Error(err))
	}()

	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (op *OPBot) QuerySyncing() (bool, error) {
	address := op.DockerOPBot.queryServerUrl + "/syncing"

	data, err := op.query(address, nil)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(string(data))
}

func (op *OPBot) QueryExecutorStatus() (executor.Status, error) {
	address := op.DockerOPBot.queryServerUrl + "/status"

	data, err := op.query(address, nil)
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

func (op *OPBot) QueryWithdrawal(sequence uint64) (executortypes.QueryWithdrawalResponse, error) {
	address := fmt.Sprintf("%s/withdrawal/%d", op.DockerOPBot.queryServerUrl, sequence)
	res, err := op.query(address, nil)
	if err != nil {
		return executortypes.QueryWithdrawalResponse{}, err
	} else if strings.Contains(string(res), "not found") {
		return executortypes.QueryWithdrawalResponse{}, errors.New("withdrawal not found")
	}

	var response executortypes.QueryWithdrawalResponse
	err = json.Unmarshal(res, &response)
	if err != nil {
		return executortypes.QueryWithdrawalResponse{}, err
	}
	return response, nil
}

func (op *OPBot) QueryWithdrawals(address string, offset uint64, limit uint64, descOrder bool) (executortypes.QueryWithdrawalsResponse, error) {
	order := "desc"
	if !descOrder {
		order = "asc"
	}

	params := map[string]string{
		"address": address,
		"offset":  strconv.FormatUint(offset, 10),
		"limit":   strconv.FormatUint(limit, 10),
		"order":   order,
	}
	url := op.DockerOPBot.queryServerUrl + "/withdrawals"
	res, err := op.query(url, params)
	if err != nil {
		return executortypes.QueryWithdrawalsResponse{}, err
	}

	var response executortypes.QueryWithdrawalsResponse
	err = json.Unmarshal(res, &response)
	if err != nil {
		return executortypes.QueryWithdrawalsResponse{}, err
	}
	return response, nil
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

func (c commander) GrantOraclePermissions(address string, homeDir string) []string {
	return []string{
		"opinitd", "tx", "grant-oracle", address,
		"--home", homeDir,
	}
}

func (commander) DefaultContainerImage() string {
	return DefaultContainerImage
}

func (commander) DefaultContainerVersion() string {
	return DefaultContainerVersion
}

func (commander) ParseAddKeyOutput(stdout, stderr, bech32Prefix string) (ibc.Wallet, error) {
	var wallet WalletModel
	err := json.Unmarshal([]byte(stdout), &wallet)

	for keyName, elem := range wallet {
		opWallet := NewWallet(keyName, elem.Address, elem.Mnemonic, bech32Prefix)
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

func (c commander) CreateWallet(keyName, address, mnemonic, bech32Prefix string) ibc.Wallet {
	return NewWallet(keyName, address, mnemonic, bech32Prefix)
}

var _ ibc.Wallet = &OPWallet{}

type WalletModel map[string]struct {
	Mnemonic string `json:"mnemonic"`
	Address  string `json:"address"`
}

type OPWallet struct {
	mnemonic     string
	address      string
	keyName      string
	bech32Prefix string
}

func NewWallet(keyname string, address string, mnemonic string, bech32Prefix string) *OPWallet {
	return &OPWallet{
		mnemonic:     mnemonic,
		address:      address,
		keyName:      keyname,
		bech32Prefix: bech32Prefix,
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
	addr, err := sdk.GetFromBech32(op.address, op.bech32Prefix)
	if err != nil {
		panic(errors.Wrap(err, "failed to decode address"))
	}
	return addr
}
