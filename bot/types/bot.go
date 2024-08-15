package types

import (
	"context"
)

type Bot interface {
	Initialize(context.Context) error
	Start(context.Context) error
	Close()
}
