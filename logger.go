package main

import (
	"fmt"
	"log"
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var once sync.Once
var logger *zap.Logger

func getGlobalLogger() *zap.Logger {
	once.Do(func() {
		logger = initLogger()
	})
	return logger
}

func initLogger() *zap.Logger {
	// Configuring encoder
	encoderConfig := zapcore.EncoderConfig{
		MessageKey:    "message",
		LevelKey:      "level",
		TimeKey:       "time",
		NameKey:       "logger",
		CallerKey:     "caller",
		FunctionKey:   zapcore.OmitKey,
		StacktraceKey: "stacktrace",
		LineEnding:    zapcore.DefaultLineEnding,
		EncodeLevel:   zapcore.CapitalColorLevelEncoder,
		EncodeTime:    zapcore.ISO8601TimeEncoder,
	}

	// Configuring console encoder
	consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)

	// Configuring file encoder
	fileEncoder := zapcore.NewJSONEncoder(encoderConfig)

	// Creating console and file write syncers
	consoleDebugging := zapcore.Lock(os.Stdout)
	file, err := os.Create("cm_controller.log")
	if err != nil {
		panic(err)
	}
	fileDebugging := zapcore.AddSync(file)

	//Setting log level
	level := zap.InfoLevel
	levelEnv := os.Getenv("LOG_LEVEL")
	if levelEnv != "" {
		levelFromEnv, err := zapcore.ParseLevel(levelEnv)
		if err != nil {
			log.Println(
				fmt.Errorf("invalid level, defaulting to INFO: %w", err),
			)
		}

		level = levelFromEnv
	}
	logLevel := zap.NewAtomicLevelAt(level)

	// Creating core
	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, consoleDebugging, logLevel),
		zapcore.NewCore(fileEncoder, fileDebugging, logLevel),
	)

	// Creating logger
	logger := zap.New(core)

	return logger
}
