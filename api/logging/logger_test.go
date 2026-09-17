package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewJSONLoggerLevels(t *testing.T) {
	for name, tt := range map[string]struct {
		level    string
		expected slog.Level
	}{
		"empty defaults to INFO":       {level: "", expected: slog.LevelInfo},
		"unparseable defaults to INFO": {level: "UNKNOWN_LEVEL", expected: slog.LevelInfo},
		"DEBUG":                        {level: "DEBUG", expected: slog.LevelDebug},
		"INFO":                         {level: "INFO", expected: slog.LevelInfo},
		"WARN":                         {level: "WARN", expected: slog.LevelWarn},
		"ERROR":                        {level: "ERROR", expected: slog.LevelError},
		"lowercase is accepted":        {level: "debug", expected: slog.LevelDebug},
	} {
		t.Run(name, func(t *testing.T) {
			logger, levelVar := NewJSONLogger(tt.level)
			require.Equal(t, tt.expected, levelVar.Level())
			assertLogLevelStrict(t, logger, tt.expected)
		})
	}
}

// NewJSONLogger is a pure constructor: it must not touch slog.Default.
func TestNewJSONLoggerDoesNotSetDefault(t *testing.T) {
	original := slog.Default()
	t.Cleanup(func() { slog.SetDefault(original) })

	logger, _ := NewJSONLogger("DEBUG")
	require.NotSame(t, logger, slog.Default())
	require.Same(t, original, slog.Default())
}

func TestSetDefaultLogger(t *testing.T) {
	original := slog.Default()
	t.Cleanup(func() { slog.SetDefault(original) })

	logger, levelVar := SetDefaultLogger("WARN")
	require.Equal(t, slog.LevelWarn, levelVar.Level())
	require.Same(t, logger, slog.Default())
}

func TestSetDefaultFromEnv(t *testing.T) {
	original := slog.Default()
	t.Cleanup(func() { slog.SetDefault(original) })

	t.Setenv("LOG_LEVEL", "DEBUG")
	logger, levelVar := SetDefaultFromEnv()
	require.Equal(t, slog.LevelDebug, levelVar.Level())
	require.Same(t, logger, slog.Default())
}

func TestSetDefaultFromEnvUnset(t *testing.T) {
	original := slog.Default()
	t.Cleanup(func() { slog.SetDefault(original) })

	require.NoError(t, os.Unsetenv("LOG_LEVEL"))
	_, levelVar := SetDefaultFromEnv()
	require.Equal(t, slog.LevelInfo, levelVar.Level())
}

// The level is held in a LevelVar so it can be changed after the logger is
// built, without rebuilding the handler.
func TestLevelVarIsLive(t *testing.T) {
	logger, levelVar := NewJSONLogger("INFO")
	require.False(t, logger.Enabled(nil, slog.LevelDebug))

	levelVar.Set(slog.LevelDebug)
	require.True(t, logger.Enabled(nil, slog.LevelDebug))
}

// The handler must emit JSON, since that is what makes fields queryable in
// DataDog. Uses the same handler options as NewJSONLogger.
func TestOutputIsJSONWithAttributes(t *testing.T) {
	var buf bytes.Buffer
	levelVar := new(slog.LevelVar)
	levelVar.Set(slog.LevelInfo)
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: levelVar}))

	logger.With(slog.String(TraceIdKey, "trace-abc")).
		Info("something happened", slog.String(DatasetNodeIdKey, "N:dataset:1234"))

	var logged map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &logged))
	require.Equal(t, "something happened", logged["msg"])
	require.Equal(t, "INFO", logged["level"])
	require.Equal(t, "trace-abc", logged[TraceIdKey])
	require.Equal(t, "N:dataset:1234", logged[DatasetNodeIdKey])
}

func assertLogLevelStrict(t *testing.T, logger *slog.Logger, expectedLevel slog.Level) {
	require.True(t, logger.Enabled(nil, expectedLevel))
	require.True(t, logger.Enabled(nil, expectedLevel+1))
	require.False(t, logger.Enabled(nil, expectedLevel-1))
}
