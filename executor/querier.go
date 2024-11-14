package executor

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/initia-labs/opinit-bots/types"
	"github.com/pkg/errors"
)

func (ex *Executor) RegisterQuerier() {
	ex.server.RegisterQuerier("/withdrawal/:sequence", func(c *fiber.Ctx) error {
		sequenceStr := c.Params("sequence")
		if sequenceStr == "" {
			return errors.New("sequence is required")
		}
		sequence, err := strconv.ParseUint(sequenceStr, 10, 64)
		if err != nil {
			return err
		}
		res, err := ex.child.QueryWithdrawal(sequence)
		if err != nil {
			return err
		}
		return c.JSON(res)
	})

	ex.server.RegisterQuerier("/withdrawals/:address", func(c *fiber.Ctx) error {
		address := c.Params("address")
		if address == "" {
			return errors.New("address is required")
		}

		offset := c.QueryInt("offset", 0)
		uoffset, err := types.SafeInt64ToUint64(int64(offset))
		if err != nil {
			return err
		}

		limit := c.QueryInt("limit", 10)
		if limit > 100 {
			limit = 100
		}

		ulimit, err := types.SafeInt64ToUint64(int64(limit))
		if err != nil {
			return err
		}

		descOrder := true
		orderStr := c.Query("order", "desc")
		if orderStr == "asc" {
			descOrder = false
		}
		res, err := ex.child.QueryWithdrawals(address, uoffset, ulimit, descOrder)
		if err != nil {
			return err
		}
		return c.JSON(res)
	})

	ex.server.RegisterQuerier("/status", func(c *fiber.Ctx) error {
		status, err := ex.GetStatus()
		if err != nil {
			return err
		}
		return c.JSON(status)
	})
}
