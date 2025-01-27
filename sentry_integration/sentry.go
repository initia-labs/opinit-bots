package sentry_integration

import (
	"fmt"
	"os"

	"github.com/certifi/gocertifi"
	"github.com/getsentry/sentry-go"

	"github.com/initia-labs/opinit-bots/types"
)

func CaptureCurrentHubException(err error, level sentry.Level) {
	CaptureException(sentry.CurrentHub(), err, level)
}

func CaptureException(hub *sentry.Hub, err error, level sentry.Level) {
	hub.WithScope(func(scope *sentry.Scope) {
		scope.SetLevel(level)
		hub.CaptureException(err)
	})
}

func StartSentryTransaction(ctx types.Context, operation, description string) (*sentry.Span, types.Context) {
	transaction := sentry.StartTransaction(ctx, operation)
	transaction.Description = description
	return transaction, ctx.WithContext(transaction.Context())
}

func StartSentrySpan(ctx types.Context, operation, description string) (*sentry.Span, types.Context) {
	span := sentry.StartSpan(ctx, operation)
	span.Description = description
	return span, ctx.WithContext(span.Context())
}

func Init(serverName string) {
	dsn := os.Getenv("SENTRY_DSN")
	env := os.Getenv("SENTRY_ENVIRONMENT")
	release := os.Getenv("SENTRY_RELEASE")

	// Read Sentry options from environment variables
	sentryClientOptions := sentry.ClientOptions{
		Dsn:                dsn,
		ServerName:         serverName,
		EnableTracing:      true,
		ProfilesSampleRate: 1,
		TracesSampleRate:   1,
		Environment:        env,
		Release:            release,
		Tags:               map[string]string{},
	}

	fmt.Printf("server_name: %s\ndsn: %s\nenv: %s\nrelease: %s\n", serverName, dsn, env, release)

	rootCAs, err := gocertifi.CACerts()
	if err != nil {
		panic(err)
	} else {
		sentryClientOptions.CaCerts = rootCAs
	}
	err = sentry.Init(sentryClientOptions)
	if err != nil {
		panic(err)
	}
}
