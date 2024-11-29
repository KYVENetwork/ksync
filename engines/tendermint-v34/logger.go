package tendermint_v34

import (
	"fmt"
	"github.com/KYVENetwork/ksync/utils"
	klogger "github.com/KYVENetwork/ksync/utils"
	"github.com/rs/zerolog"
	"github.com/tendermint/tendermint/libs/log"
)

func TmLogger() (logger log.Logger) {
	logger = KsyncTmLogger{logger: klogger.LogFormatter(utils.EngineTendermintV34)}
	return
}

type KsyncTmLogger struct {
	logger zerolog.Logger
}

func (l KsyncTmLogger) Debug(msg string, keyvals ...interface{}) {}

func (l KsyncTmLogger) Info(msg string, keyvals ...interface{}) {
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

func (l KsyncTmLogger) Error(msg string, keyvals ...interface{}) {
	logger := l.logger.Error()

	for i := 0; i < len(keyvals); i = i + 2 {
		logger = logger.Str(fmt.Sprintf("%v", keyvals[i]), fmt.Sprintf("%v", keyvals[i+1]))
	}

	logger.Msg(msg)
}

func (l KsyncTmLogger) With(keyvals ...interface{}) (logger log.Logger) {
	logger = KsyncTmLogger{logger: klogger.LogFormatter(utils.EngineTendermintV34, keyvals)}
	return
}
