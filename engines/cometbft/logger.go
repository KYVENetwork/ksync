package cometbft

import (
	"fmt"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/rs/zerolog"
	"io"
	"os"
)

func LogFormatter(keyvals ...interface{}) zerolog.Logger {
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

func CometLogger() (logger log.Logger) {
	logger = KsyncCometLogger{logger: LogFormatter("")}
	return
}

type KsyncCometLogger struct {
	logger zerolog.Logger
}

func (l KsyncCometLogger) Debug(msg string, keyvals ...interface{}) {}

func (l KsyncCometLogger) Info(msg string, keyvals ...interface{}) {
	logger := l.logger.Info()

	for i := 0; i < len(keyvals); i = i + 2 {
		logger = logger.Str(fmt.Sprintf("%v", keyvals[i]), fmt.Sprintf("%v", keyvals[i+1]))
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
	logger = KsyncCometLogger{logger: LogFormatter(keyvals)}
	return
}
