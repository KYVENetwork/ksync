package utils

import (
	"fmt"
	"github.com/KYVENetwork/ksync/flags"
	"github.com/rs/zerolog"
	"io"
	"os"
)

var (
	Logger = NewLogger(ApplicationName)
)

func NewLogger(name string, keyvals ...interface{}) zerolog.Logger {
	writer := io.MultiWriter(os.Stdout)
	customConsoleWriter := zerolog.ConsoleWriter{Out: writer}
	customConsoleWriter.FormatCaller = func(i interface{}) string {
		return fmt.Sprintf("\x1b[36m[%s]\x1b[0m", name)
	}

	loggerWith := zerolog.New(customConsoleWriter).With()

	if len(keyvals) > 1 {
		for i := 0; i < len(keyvals); i = i + 2 {
			loggerWith = loggerWith.Str(fmt.Sprintf("%v", keyvals[i]), fmt.Sprintf("%v", keyvals[i+1]))
		}
	}

	if flags.Debug {
		return loggerWith.Timestamp().Logger().Level(zerolog.DebugLevel)
	}

	return loggerWith.Timestamp().Logger().Level(zerolog.InfoLevel)
}
