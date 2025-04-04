package e2e

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"time"

	dockertypes "github.com/docker/docker/api/types"
	volumetypes "github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"

	"github.com/strangelove-ventures/interchaintest/v8/dockerutil"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"go.uber.org/zap"
)

const (
	defaultOPBotHomeDirectory = "/home/.opinit"
	OPBotLocalImage           = "opinit-bot-local-test"
)

const (
	queryServerPort = "3000/tcp"
)

var ports = nat.PortMap{
	nat.Port(queryServerPort): {},
}

type DockerOPBot struct {
	log *zap.Logger

	botName string

	c OPBotCommander

	networkID    string
	DockerClient *client.Client
	volumeName   string

	testName string

	customImage *ibc.DockerImage
	pullImage   bool

	containerLifecycle *dockerutil.ContainerLifecycle

	wallets map[string]map[string]ibc.Wallet // chainID -> keyname -> wallet

	homeDir string

	extraStartupFlags []string

	queryServerUrl string
}

const OPBotImagePrefix = "opbot-e2etest-"

type dockerLogLine struct {
	Stream      string            `json:"stream"`
	Aux         any               `json:"aux"`
	Error       string            `json:"error"`
	ErrorDetail dockerErrorDetail `json:"errorDetail"`
}

type dockerErrorDetail struct {
	Message string `json:"message"`
}

func uniqueOPBotImageName() (string, error) {
	uuid, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("failed to generate uuid %v", err)
	}
	return OPBotImagePrefix + uuid.String()[:6], nil
}

// BuildOPBotImage builds the OPBot Docker image
func BuildOPBotImage() (string, error) {
	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Join(filepath.Dir(b), "..")

	tar, err := archive.TarWithOptions(basepath, &archive.TarOptions{})
	if err != nil {
		return "", err
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", err
	}

	image, err := uniqueOPBotImageName()
	if err != nil {
		return "", err
	}

	res, err := cli.ImageBuild(context.Background(), tar, dockertypes.ImageBuildOptions{
		Dockerfile: "Dockerfile",
		Tags:       []string{image},
	})
	if err != nil {
		return "", err
	}

	defer res.Body.Close()
	handleDockerBuildOutput(res.Body)
	return image, nil
}

// DestroyOPBotImage removes the Docker image with the provided tag
func DestroyOPBotImage(image string) error {
	// Create a Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}

	// Remove the Docker image using the provided tag (uniquestr)
	_, err = cli.ImageRemove(context.Background(), image, dockertypes.ImageRemoveOptions{
		Force:         true, // Force remove the image
		PruneChildren: true, // Remove all child images
	})
	return err
}

// ignore the build output
func handleDockerBuildOutput(body io.Reader) {
	var logLine dockerLogLine

	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		logLine.Stream = ""
		logLine.Aux = nil
		logLine.Error = ""
		logLine.ErrorDetail = dockerErrorDetail{}

		line := scanner.Text()

		_ = json.Unmarshal([]byte(line), &logLine)
	}
	if logLine.Error != "" {
		fmt.Println(logLine.Error)
	}
}

func NewDockerOPBot(ctx context.Context, log *zap.Logger, botName string, testName string, cli *client.Client, networkID string, c OPBotCommander, buildLocalImage bool) (*DockerOPBot, error) {
	opbot := DockerOPBot{
		log: log,

		botName: botName,

		c: c,

		networkID:    networkID,
		DockerClient: cli,

		pullImage: false,

		testName: testName,

		wallets: map[string]map[string]ibc.Wallet{},
		homeDir: defaultOPBotHomeDirectory,
	}

	if buildLocalImage {
		image, err := BuildOPBotImage()
		if err != nil {
			return nil, err
		}

		opbot.customImage = &ibc.DockerImage{
			Repository: image,
			Version:    "",
			UIDGID:     c.DockerUser(),
		}
	}

	containerImage := opbot.ContainerImage()
	if err := opbot.pullContainerImageIfNecessary(containerImage); err != nil {
		return nil, fmt.Errorf("pulling container image %s: %w", containerImage.Ref(), err)
	}

	v, err := cli.VolumeCreate(ctx, volumetypes.CreateOptions{
		Labels: map[string]string{dockerutil.CleanupLabel: testName},
	})
	if err != nil {
		return nil, fmt.Errorf("creating volume: %w", err)
	}
	opbot.volumeName = v.Name

	if err := dockerutil.SetVolumeOwner(ctx, dockerutil.VolumeOwnerOptions{
		Log: opbot.log,

		Client: opbot.DockerClient,

		VolumeName: opbot.volumeName,
		ImageRef:   containerImage.Ref(),
		TestName:   testName,
		UidGid:     containerImage.UIDGID,
	}); err != nil {
		return nil, fmt.Errorf("set volume owner: %w", err)
	}

	if init := opbot.c.Init(botName, opbot.HomeDir()); len(init) > 0 {
		// Initialization should complete immediately,
		// but add a 1-minute timeout in case Docker hangs on a developer workstation.
		ctx, cancel := context.WithTimeout(ctx, time.Minute)
		defer cancel()

		// Using a nop reporter here because it keeps the API simpler,
		// and the init command is typically not of high interest.
		res := opbot.Exec(ctx, init, nil)
		if res.Err != nil {
			return nil, res.Err
		}
	}

	return &opbot, nil
}

// WriteFileToHomeDir writes the file at the given relative path to the volume
func (op *DockerOPBot) WriteFileToHomeDir(ctx context.Context, relativePath string, contents []byte) error {
	fw := dockerutil.NewFileWriter(op.log, op.DockerClient, op.testName)
	if err := fw.WriteFile(ctx, op.volumeName, relativePath, contents); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}

// ReadFileFromHomeDir reads the file at the given relative path from the volume
func (op *DockerOPBot) ReadFileFromHomeDir(ctx context.Context, relativePath string) ([]byte, error) {
	fr := dockerutil.NewFileRetriever(op.log, op.DockerClient, op.testName)
	bytes, err := fr.SingleFileContent(ctx, op.volumeName, relativePath)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve %s: %w", relativePath, err)
	}
	return bytes, nil
}

// ModifyTomlConfigFile modifies the TOML config file at the given relative path
func (op *DockerOPBot) ModifyTomlConfigFile(ctx context.Context, relativePath string, modification testutil.Toml) error {
	return testutil.ModifyTomlConfigFile(ctx, op.log, op.DockerClient, op.testName, op.volumeName, relativePath, modification)
}

// AddWallet stores the wallet in the bot's wallet map.
func (op *DockerOPBot) AddWallet(chainID string, wallet ibc.Wallet) {
	if _, ok := op.wallets[chainID]; !ok {
		op.wallets[chainID] = map[string]ibc.Wallet{
			wallet.KeyName(): wallet,
		}
	}
}

// GetWallet returns the wallet from the bot's wallet map.
func (op *DockerOPBot) GetWallet(chainID string, keyName string) (ibc.Wallet, bool) {
	chainWallets, ok := op.wallets[chainID]
	if !ok {
		return nil, false
	}

	wallet, ok := chainWallets[keyName]
	return wallet, ok
}

// GetWallets returns the wallets from the bot's wallet map.
func (op *DockerOPBot) GetWallets(chainID string) (map[string]ibc.Wallet, bool) {
	wallets, ok := op.wallets[chainID]
	return wallets, ok
}

// AddKey store the key in the bot's wallet and returns the wallet.
func (op *DockerOPBot) AddKey(ctx context.Context, chainID, keyName, bech32Prefix string) (ibc.Wallet, error) {
	cmd := op.c.AddKey(chainID, keyName, bech32Prefix, op.HomeDir())

	// Adding a key should be near-instantaneous, so add a 1-minute timeout
	// to detect if Docker has hung.
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	res := op.Exec(ctx, cmd, nil)
	if res.Err != nil {
		return nil, res.Err
	}

	wallet, err := op.c.ParseAddKeyOutput(string(res.Stdout), string(res.Stderr), bech32Prefix)
	if err != nil {
		return nil, err
	}

	// Add the wallet to the bot's wallet map.
	op.AddWallet(chainID, wallet)

	return wallet, nil
}

// RestoreKey restores the key from the mnemonic and adds it to the bot's wallet.
func (op *DockerOPBot) RestoreKey(ctx context.Context, chainID, keyName, bech32Prefix, mnemonic string) error {
	cmd := op.c.RestoreKey(chainID, keyName, bech32Prefix, mnemonic, op.HomeDir())

	// Restoring a key should be near-instantaneous, so add a 1-minute timeout
	// to detect if Docker has hung.
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	res := op.Exec(ctx, cmd, nil)
	if res.Err != nil {
		return res.Err
	}
	addrBytes := op.c.ParseRestoreKeyOutput(string(res.Stdout), string(res.Stderr))

	op.AddWallet(chainID, op.c.CreateWallet(keyName, addrBytes, mnemonic, bech32Prefix))
	return nil
}

// GetExtraStartupFlags returns the extra startup flags for the bot.
func (op *DockerOPBot) GetExtraStartupFlags() []string {
	return op.extraStartupFlags
}

type CosmosTx struct {
	Code int    `json:"code"`
	Data string `json:"data"`
	Hash string `json:"hash"`
	Log  string `json:"log"`
}

// GrantOraclePermissions grants oracle permissions to the oracle bridge executor.
func (op *DockerOPBot) GrantOraclePermissions(ctx context.Context, oracleBridgeExecutorAddress string) (string, error) {
	cmd := op.c.GrantOraclePermissions(oracleBridgeExecutorAddress, op.HomeDir())

	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	res := op.Exec(ctx, cmd, nil)
	if res.Err != nil {
		return "", res.Err
	}

	output := CosmosTx{}
	err := json.Unmarshal(res.Stdout, &output)
	if err != nil {
		return "", err
	}
	if output.Code != 0 {
		return "", fmt.Errorf("transaction failed with code %d: %s", output.Code, output.Log)
	}
	return output.Hash, nil
}

func (op *DockerOPBot) UpdateBatchInfo(ctx context.Context, chainType string, newBatchSubmitterAddress string) (string, error) {
	cmd := op.c.UpdateBatchInfo(chainType, newBatchSubmitterAddress, op.HomeDir())

	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	res := op.Exec(ctx, cmd, nil)
	if res.Err != nil {
		return "", res.Err
	}

	output := CosmosTx{}
	err := json.Unmarshal(res.Stdout, &output)
	if err != nil {
		return "", err
	}
	if output.Code != 0 {
		return "", fmt.Errorf("transaction failed with code %d: %s", output.Code, output.Log)
	}
	return output.Hash, nil
}

// Exec executes the command in the bot's container.
func (op *DockerOPBot) Exec(ctx context.Context, cmd []string, env []string) dockerutil.ContainerExecResult {
	job := dockerutil.NewImage(op.log, op.DockerClient, op.networkID, op.testName, op.ContainerImage().Repository, op.ContainerImage().Version)
	opts := dockerutil.ContainerOptions{
		Env:   env,
		Binds: op.Bind(),
	}
	return job.Run(ctx, cmd, opts)
}

// Start starts the bot's container.
func (op *DockerOPBot) Start(ctx context.Context) error {
	if op.containerLifecycle != nil {
		return fmt.Errorf("tried to start OPBot again without stopping first")
	}

	containerImage := op.ContainerImage()

	cmd := op.c.Start(op.botName, op.HomeDir())

	op.containerLifecycle = dockerutil.NewContainerLifecycle(op.log, op.DockerClient, op.Name())

	if err := op.containerLifecycle.CreateContainer(
		ctx, op.testName, op.networkID, containerImage, ports, "",
		op.Bind(), nil, op.Name(), cmd, nil, []string{},
	); err != nil {
		return err
	}

	err := op.containerLifecycle.StartContainer(ctx)
	if err != nil {
		return err
	}

	hostPorts, err := op.containerLifecycle.GetHostPorts(ctx, queryServerPort)
	if err != nil {
		return err
	}
	op.queryServerUrl = fmt.Sprintf("http://%s", hostPorts[0])
	return nil
}

// Stop stops the bot's container.
func (op *DockerOPBot) Stop(ctx context.Context) error {
	if op.containerLifecycle == nil {
		return nil
	}
	if err := op.containerLifecycle.StopContainer(ctx); err != nil {
		return err
	}

	stdoutBuf := new(bytes.Buffer)
	stderrBuf := new(bytes.Buffer)
	containerID := op.containerLifecycle.ContainerID()
	rc, err := op.DockerClient.ContainerLogs(ctx, containerID, dockertypes.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       "100",
	})
	if err != nil {
		return fmt.Errorf("Stop OPBot: retrieving ContainerLogs: %w", err)
	}
	defer func() { _ = rc.Close() }()

	// Logs are multiplexed into one stream; see docs for ContainerLogs.
	_, err = stdcopy.StdCopy(stdoutBuf, stderrBuf, rc)
	if err != nil {
		return fmt.Errorf("Stop OPBot: demuxing logs: %w", err)
	}
	_ = rc.Close()

	stdout := stdoutBuf.String()
	stderr := stderrBuf.String()

	c, err := op.DockerClient.ContainerInspect(ctx, containerID)
	if err != nil {
		return fmt.Errorf("Stop OPBot: inspecting container: %w", err)
	}

	op.log.Debug(
		fmt.Sprintf("Stopped docker container\nstdout:\n%s\nstderr:\n%s", stdout, stderr),
		zap.String("container_id", containerID),
		zap.String("container", c.Name),
	)

	if err := op.containerLifecycle.RemoveContainer(ctx); err != nil {
		return err
	}

	op.containerLifecycle = nil
	return nil
}

// Pause pauses the bot's container.
func (op *DockerOPBot) Pause(ctx context.Context) error {
	if op.containerLifecycle == nil {
		return fmt.Errorf("container not running")
	}
	return op.DockerClient.ContainerPause(ctx, op.containerLifecycle.ContainerID())
}

// Resume resumes the bot's container.
func (op *DockerOPBot) Resume(ctx context.Context) error {
	if op.containerLifecycle == nil {
		return fmt.Errorf("container not running")
	}
	return op.DockerClient.ContainerUnpause(ctx, op.containerLifecycle.ContainerID())
}

// ContainerImage returns the bot's container image.
func (op *DockerOPBot) ContainerImage() ibc.DockerImage {
	if op.customImage != nil {
		return *op.customImage
	}
	return ibc.DockerImage{
		Repository: op.c.DefaultContainerImage(),
		Version:    op.c.DefaultContainerVersion(),
		UIDGID:     op.c.DockerUser(),
	}
}

// pullContainerImageIfNecessary pulls the container image if necessary.
func (op *DockerOPBot) pullContainerImageIfNecessary(containerImage ibc.DockerImage) error {
	if !op.pullImage {
		return nil
	}

	rc, err := op.DockerClient.ImagePull(context.TODO(), containerImage.Ref(), dockertypes.ImagePullOptions{})
	if err != nil {
		return err
	}

	_, _ = io.Copy(io.Discard, rc)
	_ = rc.Close()
	return nil
}

func (op *DockerOPBot) Name() string {
	return dockerutil.CondenseHostName(dockerutil.SanitizeContainerName(op.c.Name() + "-" + op.botName + "-" + op.testName))
}

func (op *DockerOPBot) Bind() []string {
	return []string{op.volumeName + ":" + op.HomeDir()}
}

func (op *DockerOPBot) HomeDir() string {
	return op.homeDir
}

func (op *DockerOPBot) UseDockerNetwork() bool {
	return true
}

type OPBotCommander interface {
	Name() string

	DefaultContainerImage() string
	DefaultContainerVersion() string

	DockerUser() string

	// create wallet
	CreateWallet(keyName, address, mnemonic, bech32Prefix string) ibc.Wallet

	// parse the output of the add key command to get wallet
	ParseAddKeyOutput(stdout, stderr, bech32Prefix string) (ibc.Wallet, error)

	// parse the output of the restore key command to get the address
	ParseRestoreKeyOutput(stdout, stderr string) string

	// create command line arguments
	Init(botName, homeDir string) []string
	AddKey(chainID, keyName, bech32Prefix, homeDir string) []string
	RestoreKey(chainID, keyName, bech32Prefix, mnemonic, homeDir string) []string
	Start(botName string, homeDir string) []string
	GrantOraclePermissions(address string, homeDir string) []string
	UpdateBatchInfo(chainType string, newBatchSubmitterAddress string, homeDir string) []string
}
