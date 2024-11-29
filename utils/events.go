package utils

import (
	"fmt"
	"github.com/KYVENetwork/ksync/types"
	"github.com/google/uuid"
	"github.com/segmentio/analytics-go"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"time"
)

var (
	syncId = uuid.New().String()
	client = analytics.New(SegmentKey)
)

func getUserId() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	ksyncDir := filepath.Join(home, ".ksync")
	if _, err = os.Stat(ksyncDir); os.IsNotExist(err) {
		if err := os.Mkdir(ksyncDir, 0o755); err != nil {
			return "", err
		}
	}

	userId := uuid.New().String()

	idFile := filepath.Join(ksyncDir, "id")
	if _, err = os.Stat(idFile); os.IsNotExist(err) {
		if err := os.WriteFile(idFile, []byte(userId), 0o755); err != nil {
			return "", err
		}
	} else {
		data, err := os.ReadFile(idFile)
		if err != nil {
			return "", err
		}
		userId = string(data)
	}

	return userId, nil
}

func getContext() *analytics.Context {
	version := "local"
	build, _ := debug.ReadBuildInfo()

	if strings.TrimSpace(build.Main.Version) != "" {
		version = strings.TrimSpace(build.Main.Version)
	}

	timezone, _ := time.Now().Zone()
	locale := os.Getenv("LANG")

	return &analytics.Context{
		App: analytics.AppInfo{
			Name:    "ksync",
			Version: version,
		},
		Location: analytics.LocationInfo{},
		OS: analytics.OSInfo{
			Name: fmt.Sprintf("%s %s", runtime.GOOS, runtime.GOARCH),
		},
		Locale:   locale,
		Timezone: timezone,
	}
}

func TrackVersionEvent(optOut bool) {
	if optOut {
		return
	}

	userId, err := getUserId()
	if err != nil {
		return
	}

	err = client.Enqueue(analytics.Track{
		UserId:  userId,
		Event:   VERSION,
		Context: getContext(),
	})

	err = client.Close()
	_ = err
}

func TrackEnginesEvent(optOut bool) {
	if optOut {
		return
	}

	userId, err := getUserId()
	if err != nil {
		return
	}

	err = client.Enqueue(analytics.Track{
		UserId:  userId,
		Event:   ENGINES,
		Context: getContext(),
	})

	err = client.Close()
	_ = err
}

func TrackInfoEvent(chainId string, optOut bool) {
	if optOut {
		return
	}

	userId, err := getUserId()
	if err != nil {
		return
	}

	err = client.Enqueue(analytics.Track{
		UserId:     userId,
		Event:      INFO,
		Properties: analytics.NewProperties().Set("chain_id", chainId),
		Context:    getContext(),
	})

	err = client.Close()
	_ = err
}

func TrackBackupEvent(backupCompression string, backupKeepRecent int64, optOut bool) {
	if optOut {
		return
	}

	userId, err := getUserId()
	if err != nil {
		return
	}

	err = client.Enqueue(analytics.Track{
		UserId: userId,
		Event:  BACKUP,
		Properties: analytics.NewProperties().
			Set("backup_compression", backupCompression).
			Set("backup_keep_recent", backupKeepRecent),
		Context: getContext(),
	})

	err = client.Close()
	_ = err
}

func TrackResetEvent(optOut bool) {
	if optOut {
		return
	}

	userId, err := getUserId()
	if err != nil {
		return
	}

	err = client.Enqueue(analytics.Track{
		UserId:  userId,
		Event:   RESET,
		Context: getContext(),
	})

	err = client.Close()
	_ = err
}

func TrackPruningEvent(untilHeight int64, optOut bool) {
	if optOut {
		return
	}

	userId, err := getUserId()
	if err != nil {
		return
	}

	err = client.Enqueue(analytics.Track{
		UserId: userId,
		Event:  PRUNE,
		Properties: analytics.NewProperties().
			Set("until_height", untilHeight),
		Context: getContext(),
	})

	err = client.Close()
	_ = err
}

func TrackServeSnapshotsEvent(engine types.Engine, chainId, chainRest, storageRest string, snapshotPort int64, rpcServer bool, rpcServerPort int64, startHeight int64, pruning, keepSnapshots, debug, optOut bool) {
	if optOut {
		return
	}

	userId, err := getUserId()
	if err != nil {
		return
	}

	// TODO: get chain id
	//project, err := engine.GetChainId()
	//if err != nil {
	//	return
	//}

	currentHeight := engine.GetHeight()

	err = client.Enqueue(analytics.Track{
		UserId: userId,
		Event:  SERVE_SNAPSHOTS,
		Properties: analytics.NewProperties().
			Set("chain_id", chainId).
			Set("chain_rest", chainRest).
			Set("storage_rest", storageRest).
			Set("project", "").
			Set("current_height", currentHeight).
			Set("snapshot_port", snapshotPort).
			Set("rpc_server", rpcServer).
			Set("rpc_server_port", rpcServerPort).
			Set("start_height", startHeight).
			Set("pruning", pruning).
			Set("keep_snapshots", keepSnapshots).
			Set("debug", debug),
		Context: getContext(),
	})

	err = client.Close()
	_ = err
}

func TrackSyncStartEvent(engine types.Engine, syncType, chainId, chainRest, storageRest string, targetHeight int64, optOut bool) {
	if optOut {
		return
	}

	userId, err := getUserId()
	if err != nil {
		return
	}

	// TODO: get chain id
	//project, err := engine.GetChainId()
	//if err != nil {
	//	return
	//}

	currentHeight := engine.GetHeight()

	err = client.Enqueue(analytics.Track{
		UserId: userId,
		Event:  SYNC_STARTED,
		Properties: analytics.NewProperties().
			Set("sync_id", syncId).
			Set("sync_type", syncType).
			Set("chain_id", chainId).
			Set("chain_rest", chainRest).
			Set("storage_rest", storageRest).
			Set("project", "").
			Set("current_height", currentHeight).
			Set("target_height", targetHeight),
		Context: getContext(),
	})
	_ = err

	return
}

func TrackSyncCompletedEvent(stateSyncHeight, blocksSynced, targetHeight int64, elapsed float64, optOut bool) {
	if optOut {
		return
	}

	userId, err := getUserId()
	if err != nil {
		return
	}

	err = client.Enqueue(analytics.Track{
		UserId: userId,
		Event:  SYNC_COMPLETED,
		Properties: analytics.NewProperties().
			Set("sync_id", syncId).
			Set("state_sync_height", stateSyncHeight).
			Set("blocks_synced", blocksSynced).
			Set("target_height", targetHeight).
			Set("duration", math.Floor(elapsed*100)/100),
		Context: getContext(),
	})

	err = client.Close()
	_ = err
}
