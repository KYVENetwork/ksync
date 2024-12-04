package metrics

import (
	"fmt"
	"github.com/KYVENetwork/ksync/flags"
	"github.com/KYVENetwork/ksync/logger"
	"github.com/google/uuid"
	"github.com/segmentio/analytics-go"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"runtime"
	runtimeDebug "runtime/debug"
	"strings"
	"time"
)

const (
	SegmentWriteKey = "aTXiSVqhrmavVNbF2M61pyBDF4stWHgf"
)

var (
	startTime                = time.Now()
	sourceId                 string
	userConfirmationInput    string
	userConfirmationDuration time.Duration
	continuationHeight       int64
	snapshotHeight           int64
	latestHeight             int64
	successfulRequests       int64
	failedRequests           int64
)

func SetSourceId(_sourceId string) {
	sourceId = _sourceId
}

func SetUserConfirmationInput(_userConfirmationInput string) {
	userConfirmationInput = _userConfirmationInput
}

func SetUserConfirmationDuration(_userConfirmationDuration time.Duration) {
	userConfirmationDuration = _userConfirmationDuration
}

func SetContinuationHeight(_continuationHeight int64) {
	continuationHeight = _continuationHeight
}

func SetSnapshotHeight(_snapshotHeight int64) {
	snapshotHeight = _snapshotHeight
}

func SetLatestHeight(_latestHeight int64) {
	latestHeight = _latestHeight
}

func IncreaseSuccessfulRequests() {
	successfulRequests++
}

func IncreaseFailedRequests() {
	failedRequests++
}

// GetSyncDuration gets the sync time duration.
// We subtract the user confirmation duration
// since this time was not spent on actually syncing the node
func GetSyncDuration() time.Duration {
	return time.Since(startTime.Add(userConfirmationDuration))
}

func getVersion() string {
	version, ok := runtimeDebug.ReadBuildInfo()
	if !ok {
		panic("failed to get ksync version")
	}

	if version.Main.Version == "" {
		return "dev"
	}

	return strings.TrimSpace(version.Main.Version)
}

func getUserId() (string, error) {
	// we identify a KSYNC user by their local id file which is stored
	// under "$HOME/.ksync/id". If the id file was not created yet
	// or got deleted we create a new one
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	ksyncDir := filepath.Join(home, ".ksync")

	// if ksync home directory does not exist we create it
	if _, err = os.Stat(ksyncDir); os.IsNotExist(err) {
		if err := os.Mkdir(ksyncDir, 0o755); err != nil {
			return "", err
		}
	}

	idFile := filepath.Join(ksyncDir, "id")

	// if id file does not exist we create a new user id and store it
	if _, err = os.Stat(idFile); os.IsNotExist(err) {
		newUserId := uuid.New().String()

		if err := os.WriteFile(idFile, []byte(newUserId), 0o755); err != nil {
			return "", err
		}

		return newUserId, nil
	}

	// if id file exists read the contents
	data, err := os.ReadFile(idFile)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func getContext() *analytics.Context {
	timezone, _ := time.Now().Zone()
	locale := os.Getenv("LANG")

	return &analytics.Context{
		App: analytics.AppInfo{
			Name:    "ksync",
			Version: getVersion(),
		},
		Location: analytics.LocationInfo{},
		OS: analytics.OSInfo{
			Name: fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH),
		},
		Locale:   locale,
		Timezone: timezone,
	}
}

func getProperties(runtimeError error) analytics.Properties {
	properties := analytics.NewProperties()

	// set flag properties (all flags must start with "flag_"
	properties.Set("flag_binary_path", flags.BinaryPath)
	properties.Set("flag_home_path", flags.HomePath)
	properties.Set("flag_chain_id", flags.ChainId)
	properties.Set("flag_chain_rest", flags.ChainRest)
	properties.Set("flag_storage_rest", flags.StorageRest)
	properties.Set("flag_block_rpc", flags.BlockRpc)
	properties.Set("flag_snapshot_pool_id", flags.SnapshotPoolId)
	properties.Set("flag_block_pool_id", flags.BlockPoolId)
	properties.Set("flag_start_height", flags.StartHeight)
	properties.Set("flag_target_height", flags.TargetHeight)
	properties.Set("flag_rpc_server", flags.RpcServer)
	properties.Set("flag_rpc_server_port", flags.RpcServerPort)
	properties.Set("flag_snapshot_port", flags.SnapshotPort)
	properties.Set("flag_block_rpc_req_timeout", flags.BlockRpcReqTimeout)
	properties.Set("flag_pruning", flags.Pruning)
	properties.Set("flag_keep_snapshots", flags.KeepSnapshots)
	properties.Set("flag_skip_waiting", flags.SkipWaiting)
	properties.Set("flag_app_logs", flags.AppLogs)
	properties.Set("flag_auto_select_binary_version", flags.AutoSelectBinaryVersion)
	properties.Set("flag_keep_addr_book", flags.KeepAddrBook)
	properties.Set("flag_opt_out", flags.OptOut)
	properties.Set("flag_debug", flags.Debug)
	properties.Set("flag_y", flags.Y)

	// set metric properties (all flags must start with "metric_"
	properties.Set("metrics_total_duration", time.Since(startTime).Milliseconds())
	properties.Set("metrics_source_id", sourceId)
	properties.Set("metrics_user_confirmation_input", userConfirmationInput)
	properties.Set("metrics_user_confirmation_duration", userConfirmationDuration.Milliseconds())
	properties.Set("metrics_continuation_height", continuationHeight)
	properties.Set("metrics_snapshot_height", snapshotHeight)
	properties.Set("metrics_latest_height", latestHeight)
	properties.Set("metrics_sync_duration", GetSyncDuration().Milliseconds())
	properties.Set("metrics_successful_requests", successfulRequests)
	properties.Set("metrics_failed_requests", failedRequests)

	if latestHeight > continuationHeight-1 {
		properties.Set("metrics_blocks_synced", latestHeight-(continuationHeight-1))
	} else {
		properties.Set("metrics_blocks_synced", 0)
	}

	// set error properties
	if runtimeError == nil {
		properties.Set("runtime_error", "")
	} else {
		properties.Set("runtime_error", runtimeError.Error())
	}

	return properties
}

func Send(cmd *cobra.Command, runtimeError error) {
	// if the user opts out we return immediately
	if flags.OptOut {
		logger.Logger.Debug().Msg("opting-out of metric collection")
		return
	}

	userId, err := getUserId()
	if err != nil {
		logger.Logger.Debug().Err(err).Msg("failed to get user id")
		return
	}

	event := "ksync"
	if cmd != nil {
		event = cmd.Use
	}

	message := analytics.Track{
		UserId:     userId,
		Event:      event,
		Context:    getContext(),
		Properties: getProperties(runtimeError),
	}

	client := analytics.New(SegmentWriteKey)
	defer client.Close()

	if err := client.Enqueue(message); err != nil {
		logger.Logger.Debug().Err(err).Msg("failed to enqueue track message")
		return
	}

	logger.Logger.Debug().Str("event", message.Event).Str("userId", message.UserId).Any("context", message.Context).Any("properties", message.Properties).Msg("sent track message")
}
