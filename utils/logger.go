package utils

import (
	"fmt"
	"github.com/rs/zerolog"
	"io"
	"os"
)

func KsyncLogger(moduleName string) zerolog.Logger {
	writer := io.MultiWriter(os.Stdout)
	customConsoleWriter := zerolog.ConsoleWriter{Out: writer}
	customConsoleWriter.FormatCaller = func(i interface{}) string {
		return "\x1b[36m[KSYNC]\x1b[0m"
	}

	logger := zerolog.New(customConsoleWriter).With().Str("module", moduleName).Timestamp().Logger()
	return logger
}

func LogFormatter(name string, keyvals ...interface{}) zerolog.Logger {
	writer := io.MultiWriter(os.Stdout)
	customConsoleWriter := zerolog.ConsoleWriter{Out: writer}
	customConsoleWriter.FormatCaller = func(i interface{}) string {
		return fmt.Sprintf("\x1b[36m[%s]\x1b[0m", name)
	}

	logger := zerolog.New(customConsoleWriter).With()

	if len(keyvals) > 1 {
		for i := 0; i < len(keyvals); i = i + 2 {
			logger = logger.Str(fmt.Sprintf("%v", keyvals[i]), fmt.Sprintf("%v", keyvals[i+1]))
		}
	}

	return logger.Timestamp().Logger()
}
