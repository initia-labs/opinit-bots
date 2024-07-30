package types

type BotType string

const (
	BotTypeExecutor BotType = "executor"
)

func BotTypeFromString(name string) BotType {
	switch name {
	case "executor":
		return BotTypeExecutor
	}

	panic("unknown bot type")
}
