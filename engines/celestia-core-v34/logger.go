package celestia_core_v34

import (
	"fmt"
	"github.com/KYVENetwork/celestia-core/libs/log"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/rs/zerolog"
)

var (
	engineLogger = EngineLogger{logger: utils.NewLogger(utils.EngineCelestiaCoreV34)}
)

type EngineLogger struct {
	logger zerolog.Logger
}

func (l EngineLogger) Debug(msg string, keyvals ...interface{}) {
	logger := l.logger.Debug()

	for i := 0; i < len(keyvals); i = i + 2 {
		logger = logger.Str(fmt.Sprintf("%v", keyvals[i]), fmt.Sprintf("%v", keyvals[i+1]))
	}

	logger.Msg(msg)
}

func (l EngineLogger) Info(msg string, keyvals ...interface{}) {
	logger := l.logger.Info()

	for i := 0; i < len(keyvals); i = i + 2 {
		if keyvals[i] == "hash" || keyvals[i] == "appHash" {
			logger = logger.Str(fmt.Sprintf("%v", keyvals[i]), fmt.Sprintf("%X", keyvals[i+1]))
		} else {
			logger = logger.Str(fmt.Sprintf("%v", keyvals[i]), fmt.Sprintf("%v", keyvals[i+1]))
		}
	}

	logger.Msg(msg)
}

func (l EngineLogger) Error(msg string, keyvals ...interface{}) {
	logger := l.logger.Error()

	for i := 0; i < len(keyvals); i = i + 2 {
		logger = logger.Str(fmt.Sprintf("%v", keyvals[i]), fmt.Sprintf("%v", keyvals[i+1]))
	}

	logger.Msg(msg)
}

func (l EngineLogger) With(keyvals ...interface{}) (logger log.Logger) {
	logger = EngineLogger{logger: utils.NewLogger(utils.EngineCelestiaCoreV34, keyvals)}
	return
}
