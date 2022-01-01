package gkeutil

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger configured with datetime, caller information,
// and splits output to stdout and stderr based on error level.
var Logger *zap.Logger

func init() {
	zap.NewProduction()
	isErrorLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.ErrorLevel
	})
	isInfoLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl < zapcore.ErrorLevel
	})
	stdoutWriter := zapcore.Lock(os.Stdout)
	stderrWriter := zapcore.Lock(os.Stderr)

	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.RFC3339TimeEncoder
	encoder := zapcore.NewJSONEncoder(config)

	core := zapcore.NewTee(
		zapcore.NewCore(encoder, stderrWriter, isErrorLevel),
		zapcore.NewCore(encoder, stdoutWriter, isInfoLevel),
	)
	Logger = zap.New(core, zap.AddCaller())
}
