package logger

import (
	"github.com/tendermint/tendermint/libs/log"
	"os"
)

func Logger() log.Logger {
	return log.NewTMLogger(log.NewSyncWriter(os.Stdout))
}
