package logger

import (
	"github.com/tendermint/tendermint/libs/log"
	"os"
)

func Logger() log.Logger {
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))
	option, err := log.AllowLevel("info")
	if err != nil {
		panic(err)
	}
	return log.NewFilter(logger, option)
}
