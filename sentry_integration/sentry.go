package sentry_integration

import (
	"fmt"
	"os"
	"strconv"

	"github.com/certifi/gocertifi"
	"github.com/getsentry/sentry-go"

	"github.com/initia-labs/opinit-bots/types"
)

const (
	DEFAULT_PROFILES_SAMPLE_RATE = 0.01
	DEFAULT_TRACES_SAMPLE_RATE   = 0.01
)

func CaptureCurrentHubException(err error, level sentry.Level) {
	CaptureException(sentry.CurrentHub(), err, level)
}

func CaptureCurrentHubMessage(message string, level sentry.Level) {
	CaptureMessage(sentry.CurrentHub(), message, level)
}

func CaptureException(hub *sentry.Hub, err error, level sentry.Level) {
	hub.WithScope(func(scope *sentry.Scope) {
		scope.SetLevel(level)
		hub.CaptureException(err)
	})
}

func CaptureMessage(hub *sentry.Hub, message string, level sentry.Level) {
	hub.WithScope(func(scope *sentry.Scope) {
		scope.SetLevel(level)
		hub.CaptureMessage(message)
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

func Init(serverName, component string) error {
	dsn := os.Getenv("SENTRY_DSN")
	if dsn == "" {
		fmt.Println("SENTRY_DSN is not set, skipping sentry initialization")
		return nil
	}

	env := os.Getenv("SENTRY_ENVIRONMENT")
	if env == "" {
		return fmt.Errorf("SENTRY_ENVIRONMENT is not set")
	}

	release := os.Getenv("SENTRY_RELEASE")
	if release == "" {
		return fmt.Errorf("SENTRY_RELEASE is not set")
	}

	l1Chain := os.Getenv("SENTRY_L1_CHAIN")
	if l1Chain == "" {
		return fmt.Errorf("SENTRY_L1_CHAIN is not set")
	}

	l2Chain := os.Getenv("SENTRY_L2_CHAIN")
	if l2Chain == "" {
		return fmt.Errorf("SENTRY_L2_CHAIN is not set")
	}

	profilesSampleRate := DEFAULT_PROFILES_SAMPLE_RATE
	tracesSampleRate := DEFAULT_TRACES_SAMPLE_RATE

	if rate := os.Getenv("SENTRY_PROFILES_SAMPLE_RATE"); rate != "" {
		if parsed, err := strconv.ParseFloat(rate, 64); err == nil {
			profilesSampleRate = parsed
		}
	}
	if rate := os.Getenv("SENTRY_TRACES_SAMPLE_RATE"); rate != "" {
		if parsed, err := strconv.ParseFloat(rate, 64); err == nil {
			tracesSampleRate = parsed
		}
	}

	// Read Sentry options from environment variables
	sentryClientOptions := sentry.ClientOptions{
		Dsn:                dsn,
		ServerName:         serverName,
		EnableTracing:      true,
		ProfilesSampleRate: profilesSampleRate,
		TracesSampleRate:   tracesSampleRate,
		Environment:        env,
		Release:            release,
		Tags: map[string]string{
			"l1_chain":    l1Chain,
			"l2_chain":    l2Chain,
			"environment": env,
			"component":   component,
			"commit_sha":  release,
		},
	}

	rootCAs, err := gocertifi.CACerts()
	if err != nil {
		return err
	} else {
		sentryClientOptions.CaCerts = rootCAs
	}
	err = sentry.Init(sentryClientOptions)
	if err != nil {
		return fmt.Errorf("failed to initialize sentry: %w", err)
	}
	fmt.Printf("sentry initialized for %s\n", serverName)
	return nil
}
