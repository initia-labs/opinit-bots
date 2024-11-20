package types

import (
	"github.com/initia-labs/opinit-bots/types"
)

type Bot interface {
	Initialize(types.Context) error
	Start(types.Context) error
}
