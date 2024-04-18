package celestia_core_v34

import (
	"fmt"
	"github.com/KYVENetwork/celestia-core/libs/log"
	klogger "github.com/KYVENetwork/ksync/utils"
	"github.com/rs/zerolog"
)

func TmLogger() (logger log.Logger) {
	logger = KsyncTmLogger{logger: klogger.LogFormatter("")}
	return
}

type KsyncTmLogger struct {
	logger zerolog.Logger
}

func (l KsyncTmLogger) Debug(msg string, keyvals ...interface{}) {}

func (l KsyncTmLogger) Info(msg string, keyvals ...interface{}) {
	logger := l.logger.Info()

	for i := 0; i < len(keyvals); i = i + 2 {
		logger = logger.Str(fmt.Sprintf("%v", keyvals[i]), fmt.Sprintf("%v", keyvals[i+1]))
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
	logger = KsyncTmLogger{logger: klogger.LogFormatter(keyvals)}
	return
}
