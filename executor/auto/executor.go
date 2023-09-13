package auto

import (
	"github.com/KYVENetwork/ksync/executor/db"
	"github.com/KYVENetwork/ksync/executor/p2p"
	"github.com/KYVENetwork/ksync/types"
)

func InitExecutor(quitCh chan<- int) []*types.SyncProcess {
	processes := []*types.SyncProcess{
		{Name: "p2p", Goroutine: nil, QuitCh: quitCh, Running: false},
		{Name: "db", Goroutine: nil, QuitCh: quitCh, Running: false},
	}

	return processes
}

func StartSyncProcess(s *types.SyncProcess, home string, poolId int64, restEndpoint string, targetHeight int64, apiServer bool, port int64) {
	if !s.Running {
		s.Running = true
		if s.Name == "p2p" {
			logger.Info().Msg("starting P2P syncing process")
			go p2p.StartP2PExecutor(home, poolId, restEndpoint)
		} else if s.Name == "db" {
			logger.Info().Msg("starting DB syncing process")
			go db.StartDBExecutor(home, restEndpoint, poolId, targetHeight, apiServer, port)
		}
	}
}

func StopProcess(s *types.SyncProcess) {
	if s.Running {
		close(s.QuitCh)
		s.Running = false
	}
}
