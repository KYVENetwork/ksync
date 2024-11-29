package cometbft_v38

import (
	"fmt"
	"github.com/KYVENetwork/cometbft/v38/libs/log"
	"github.com/KYVENetwork/ksync/utils"
	klogger "github.com/KYVENetwork/ksync/utils"
	"github.com/rs/zerolog"
)

func CometLogger() (logger log.Logger) {
	logger = KsyncCometLogger{logger: klogger.LogFormatter(utils.EngineCometBFTV38)}
	return
}

type KsyncCometLogger struct {
	logger zerolog.Logger
}

func (l KsyncCometLogger) Debug(msg string, keyvals ...interface{}) {}

func (l KsyncCometLogger) Info(msg string, keyvals ...interface{}) {
	logger := l.logger.Info()

	for i := 0; i < len(keyvals); i = i + 2 {
		if keyvals[i] == "hash" || keyvals[i] == "appHash" {
			logger = logger.Str(fmt.Sprintf("%v", keyvals[i]), fmt.Sprintf("%x", keyvals[i+1]))
		} else {
			logger = logger.Str(fmt.Sprintf("%v", keyvals[i]), fmt.Sprintf("%v", keyvals[i+1]))
		}
	}

	logger.Msg(msg)
}

func (l KsyncCometLogger) Error(msg string, keyvals ...interface{}) {
	logger := l.logger.Error()

	for i := 0; i < len(keyvals); i = i + 2 {
		logger = logger.Str(fmt.Sprintf("%v", keyvals[i]), fmt.Sprintf("%v", keyvals[i+1]))
	}

	logger.Msg(msg)
}

func (l KsyncCometLogger) With(keyvals ...interface{}) (logger log.Logger) {
	logger = KsyncCometLogger{logger: klogger.LogFormatter(utils.EngineCometBFTV38, keyvals)}
	return
}
