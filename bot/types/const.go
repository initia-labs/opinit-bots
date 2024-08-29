package types

import (
	"errors"
	"fmt"
)

type BotType string

const (
	BotTypeExecutor   BotType = "executor"
	BotTypeChallenger BotType = "challenger"
)

func (b BotType) Validate() error {
	if b != BotTypeExecutor && b != BotTypeChallenger {
		return errors.New(fmt.Sprintf("invalid bot type: %s", b))
	}
	return nil
}

func BotTypeFromString(name string) BotType {
	switch name {
	case "executor":
		return BotTypeExecutor
	case "challenger":
		return BotTypeChallenger
	}
	panic("unknown bot type")
}
