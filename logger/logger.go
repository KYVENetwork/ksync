package logger

import (
	"github.com/rs/zerolog"
	"github.com/tendermint/tendermint/libs/log"
	"io"
	"os"
)

func KLogger() log.Logger {
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))
	option, err := log.AllowLevel("info")
	if err != nil {
		panic(err)
	}
	return log.NewFilter(logger, option)
}

func Logger(moduleName string) zerolog.Logger {
	writer := io.MultiWriter(os.Stdout)
	customConsoleWriter := zerolog.ConsoleWriter{Out: writer}
	customConsoleWriter.FormatCaller = func(i interface{}) string {
		return "\x1b[36m[KSYNC]\x1b[0m"
	}

	logger := zerolog.New(customConsoleWriter).With().Str("module", moduleName).Timestamp().Logger()
	return logger
}
