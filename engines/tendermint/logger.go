package tendermint

import (
	"fmt"
	"github.com/rs/zerolog"
	"github.com/tendermint/tendermint/libs/log"
	"io"
	"os"
)

func KLogger() (logger log.Logger) {
	logger = KsyncTmLogger{logger: TmLogger("")}
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
	logger = KsyncTmLogger{logger: TmLogger(keyvals)}
	return
}

func KsyncLogger(moduleName string) zerolog.Logger {
	writer := io.MultiWriter(os.Stdout)
	customConsoleWriter := zerolog.ConsoleWriter{Out: writer}
	customConsoleWriter.FormatCaller = func(i interface{}) string {
		return "\x1b[36m[KSYNC]\x1b[0m"
	}

	logger := zerolog.New(customConsoleWriter).With().Str("module", moduleName).Timestamp().Logger()
	return logger
}

func TmLogger(keyvals ...interface{}) zerolog.Logger {
	writer := io.MultiWriter(os.Stdout)
	customConsoleWriter := zerolog.ConsoleWriter{Out: writer}
	customConsoleWriter.FormatCaller = func(i interface{}) string {
		return "\x1b[36m[APP]\x1b[0m"
	}

	logger := zerolog.New(customConsoleWriter).With()

	if len(keyvals) > 1 {
		for i := 0; i < len(keyvals); i = i + 2 {
			logger = logger.Str(fmt.Sprintf("%v", keyvals[i]), fmt.Sprintf("%v", keyvals[i+1]))
		}
	}

	return logger.Timestamp().Logger()
}
