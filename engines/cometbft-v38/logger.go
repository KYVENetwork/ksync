package cometbft_v38

import (
	"fmt"
	"github.com/KYVENetwork/cometbft/v38/libs/log"
	"github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/rs/zerolog"
)

var (
	engineLogger = EngineLogger{logger: logger.NewLogger(utils.EngineCometBFTV38)}
)

type EngineLogger struct {
	logger zerolog.Logger
}

func (l EngineLogger) Debug(msg string, keyvals ...interface{}) {
	logger := l.logger.Debug()

	for i := 0; i < len(keyvals); i = i + 2 {
		logger = logger.Any(fmt.Sprintf("%s", keyvals[i]), keyvals[i+1])
	}

	logger.Msg(msg)
}

func (l EngineLogger) Info(msg string, keyvals ...interface{}) {
	logger := l.logger.Info()

	for i := 0; i < len(keyvals); i = i + 2 {
		if keyvals[i] == "hash" || keyvals[i] == "appHash" {
			logger = logger.Str(fmt.Sprintf("%v", keyvals[i]), fmt.Sprintf("%s", keyvals[i+1]))
		} else {
			logger = logger.Any(fmt.Sprintf("%s", keyvals[i]), keyvals[i+1])
		}
	}

	logger.Msg(msg)
}

func (l EngineLogger) Error(msg string, keyvals ...interface{}) {
	logger := l.logger.Error()

	for i := 0; i < len(keyvals); i = i + 2 {
		logger = logger.Any(fmt.Sprintf("%s", keyvals[i]), keyvals[i+1])
	}

	logger.Msg(msg)
}

func (l EngineLogger) With(keyvals ...interface{}) log.Logger {
	return EngineLogger{logger: logger.NewLogger(utils.EngineCometBFTV38, keyvals)}
}
