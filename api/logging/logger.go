// Package logging configures the service's structured logger (log/slog): a JSON
// handler writing to stdout at a configurable level, with a testable pure
// constructor (NewJSONLogger) separated from the side-effecting
// SetDefaultLogger. Replaces sirupsen/logrus.
//
// This package previously installed the default logger from its own init(),
// which competed with near-identical init() blocks in the handler and store
// packages. Go runs package init() functions in dependency/import order, so
// "whichever runs last wins" decided the effective log level. The entrypoint
// now calls SetDefaultFromEnv explicitly instead.
package logging

import (
	"log/slog"
	"os"
)

// SetDefaultLogger builds a JSON logger at the given level and installs it as
// slog.Default. Returns the logger and its LevelVar (so the level can be changed
// at runtime). An unparseable level falls back to INFO.
func SetDefaultLogger(level string) (*slog.Logger, *slog.LevelVar) {
	logger, levelVar := NewJSONLogger(level)
	slog.SetDefault(logger)
	slog.Debug("log level set", slog.String(LevelKey, levelVar.String()))
	return logger, levelVar
}

// SetDefaultFromEnv installs the default logger using LOG_LEVEL from the
// environment (INFO when unset/invalid). Convenience wrapper for the Lambda
// entrypoint, whose deployment already sets LOG_LEVEL.
func SetDefaultFromEnv() (*slog.Logger, *slog.LevelVar) {
	return SetDefaultLogger(os.Getenv("LOG_LEVEL"))
}

// NewJSONLogger returns a JSON-handler slog.Logger at the given level (INFO if
// the string is empty or unparseable) plus its LevelVar. Pure constructor: no
// global side effects, so tests can build isolated loggers.
func NewJSONLogger(level string) (*slog.Logger, *slog.LevelVar) {
	var logLevel slog.Level
	if err := logLevel.UnmarshalText([]byte(level)); err != nil {
		if level != "" {
			slog.Error("error unmarshalling log level value",
				slog.String(LevelKey, level),
				slog.Any(ErrorKey, err))
		}
		logLevel = slog.LevelInfo
	}

	levelVar := new(slog.LevelVar)
	levelVar.Set(logLevel)

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: levelVar})
	return slog.New(handler), levelVar
}
