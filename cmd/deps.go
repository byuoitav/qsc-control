package main

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func buildLogger(lvl string) (*zap.Logger, *zap.AtomicLevel) {
	level, err := getZapLevelFromString(lvl)
	if err != nil {
		panic(fmt.Sprintf("cannot build logger: %s", err))
	}

	atom := zap.NewAtomicLevelAt(level)

	config := zap.Config{
		Level: atom,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding: "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "@",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "trace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}

	log, err := config.Build()
	if err != nil {
		panic(fmt.Sprintf("unable to build logger: %s", err))
	}

	return log, &atom
}

func getZapLevelFromString(lvl string) (zapcore.Level, error) {
	var level zapcore.Level
	if err := level.Set(lvl); err != nil {
		return level, fmt.Errorf("invalid log level: %s", err)
	}

	return level, nil
}
