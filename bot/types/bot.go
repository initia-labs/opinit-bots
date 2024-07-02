package types

import (
	"context"
)

type Bot interface {
	Start(context.Context) error
}
