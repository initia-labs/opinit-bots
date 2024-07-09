package types

type ErrorLevel uint8

const (
	ErrorLevelShutdown ErrorLevel = iota
	ErrorLevelRetry
	ErrorLevelIgnore
)

type KnownErrors map[error]ErrorLevel
