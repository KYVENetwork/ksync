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
	userId    = uuid.New().String()
	sessionId = uuid.New().String()
	client    = analytics.New(SegmentKey)
)

func TrackCmdStartEvent(command string, optOut bool) {
	if optOut {
		return
	}

	version := "local"
	build, _ := debug.ReadBuildInfo()

	if strings.TrimSpace(build.Main.Version) != "" {
		version = strings.TrimSpace(build.Main.Version)
	}

	timezone, _ := time.Now().Zone()
	locale := os.Getenv("LANG")

	err := client.Enqueue(analytics.Track{
		UserId: userId,
		Event:  command,
		Context: &analytics.Context{
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
		},
	})

	err = client.Close()
	_ = err
}

func TrackSyncStartEvent(engine types.Engine, command, chainId, chainRest, storageRest string, targetHeight int64, optOut bool) {
	if optOut {
		return
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	ksyncDir := filepath.Join(home, ".ksync")
	if _, err = os.Stat(ksyncDir); os.IsNotExist(err) {
		if err := os.Mkdir(ksyncDir, 0o755); err != nil {
			return
		}
	}

	idFile := filepath.Join(ksyncDir, "id")
	if _, err = os.Stat(idFile); os.IsNotExist(err) {
		if err := os.WriteFile(idFile, []byte(userId), 0o755); err != nil {
			return
		}
	} else {
		data, err := os.ReadFile(idFile)
		if err != nil {
			return
		}
		userId = string(data)
	}

	version := "local"
	build, _ := debug.ReadBuildInfo()

	if strings.TrimSpace(build.Main.Version) != "" {
		version = strings.TrimSpace(build.Main.Version)
	}

	timezone, _ := time.Now().Zone()
	locale := os.Getenv("LANG")

	project, err := engine.GetChainId()
	if err != nil {
		return
	}

	currentHeight := engine.GetHeight()

	err = client.Enqueue(analytics.Track{
		UserId: userId,
		Event:  command,
		Properties: analytics.NewProperties().
			Set("session_id", sessionId).
			Set("chain_id", chainId).
			Set("chain_rest", chainRest).
			Set("storage_rest", storageRest).
			Set("project", project).
			Set("current_height", currentHeight).
			Set("target_height", targetHeight),
		Context: &analytics.Context{
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
		},
	})
	_ = err

	return
}

func TrackSyncCompletedEvent(stateSyncHeight, blocksSynced, targetHeight int64, elapsed float64, optOut bool) {
	if optOut {
		return
	}

	version := "local"
	build, _ := debug.ReadBuildInfo()

	if strings.TrimSpace(build.Main.Version) != "" {
		version = strings.TrimSpace(build.Main.Version)
	}

	timezone, _ := time.Now().Zone()
	locale := os.Getenv("LANG")

	err := client.Enqueue(analytics.Track{
		UserId: userId,
		Event:  SYNC_COMPLETED,
		Properties: analytics.NewProperties().
			Set("session_id", sessionId).
			Set("state_sync_height", stateSyncHeight).
			Set("blocks_synced", blocksSynced).
			Set("target_height", targetHeight).
			Set("duration", math.Floor(elapsed*100)/100),
		Context: &analytics.Context{
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
		},
	})

	err = client.Close()
	_ = err
}
