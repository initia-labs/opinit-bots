package executor

import (
	"context"
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/initia-labs/opinit-bots-go/executor/child"
	"github.com/initia-labs/opinit-bots-go/executor/host"
	"github.com/initia-labs/opinit-bots-go/server"

	bottypes "github.com/initia-labs/opinit-bots-go/bot/types"
	executortypes "github.com/initia-labs/opinit-bots-go/executor/types"
	"github.com/initia-labs/opinit-bots-go/types"
	"go.uber.org/zap"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
)

var _ bottypes.Bot = &Executor{}

type Executor struct {
	host  *host.Host
	child *child.Child

	cfg    *executortypes.Config
	db     types.DB
	server *server.Server
	logger *zap.Logger
}

func NewExecutor(cfg *executortypes.Config, db types.DB, sv *server.Server, logger *zap.Logger, cdc codec.Codec, txConfig client.TxConfig) *Executor {
	err := cfg.Validate()
	if err != nil {
		panic(err)
	}

	h := &host.Host{}
	ch := &child.Child{}

	executor := &Executor{
		host:  h,
		child: ch,

		cfg:    cfg,
		db:     db,
		server: sv,
		logger: logger,
	}

	*h = *host.NewHost(cfg.Version, cfg.HostNode, db.WithPrefix([]byte(executortypes.HostNodeName)), logger.Named(executortypes.HostNodeName), cdc, txConfig, ch)
	*ch = *child.NewChild(cfg.Version, cfg.ChildNode, db.WithPrefix([]byte(executortypes.ChildNodeName)), logger.Named(executortypes.ChildNodeName), cdc, txConfig, h)

	bridgeInfo, err := ch.QueryBridgeInfo()
	if err != nil {
		panic(err)
	}
	if bridgeInfo.BridgeId == 0 {
		panic("bridge info is not set")
	}
	executor.logger.Info("bridge info", zap.Uint64("id", bridgeInfo.BridgeId))

	ch.SetBridgeInfo(bridgeInfo)
	h.SetBridgeId(int64(bridgeInfo.BridgeId))

	ch.RegisterHandlers()
	h.RegisterHandlers()

	executor.RegisterQuerier()
	return executor
}

func (ex *Executor) Start(cmdCtx context.Context) error {
	defer ex.db.Close()

	err := ex.server.ShutdownWithContext(cmdCtx)
	if err != nil {
		return err
	}

	hostCtx, hostDone := context.WithCancel(cmdCtx)
	childCtx, childDone := context.WithCancel(cmdCtx)

	ex.host.Start(hostCtx)
	ex.child.Start(childCtx)

	err = ex.server.Start()
	// TODO: safely shut down
	hostDone()
	childDone()
	return err
}

func (ex *Executor) RegisterQuerier() {
	ex.server.RegisterQuerier("/proofs/:sequence", func(c *fiber.Ctx) error {
		sequenceStr := c.Params("sequence")
		if sequenceStr == "" {
			return errors.New("sequence is required")
		}
		sequence, err := strconv.ParseUint(sequenceStr, 10, 64)
		if err != nil {
			return err
		}

		proofs, outputIndex, outputRoot, latestBlockHash, err := ex.child.QueryProofs(sequence)
		if err != nil {
			return err
		}

		return c.JSON(executortypes.QueryProofsResponse{
			WithdrawalProofs: proofs,
			OutputIndex:      outputIndex,
			StorageRoot:      outputRoot,
			LatestBlockHash:  latestBlockHash,
		})
	})
}
