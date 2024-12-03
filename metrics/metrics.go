package metrics

import (
	"fmt"
	"github.com/KYVENetwork/ksync/flags"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/google/uuid"
	"github.com/segmentio/analytics-go"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

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
			Version: utils.GetVersion(),
		},
		Location: analytics.LocationInfo{},
		OS: analytics.OSInfo{
			Name: fmt.Sprintf("%s %s", runtime.GOOS, runtime.GOARCH),
		},
		Locale:   locale,
		Timezone: timezone,
	}
}

func getProperties(startTime time.Time, runtimeError error) analytics.Properties {
	properties := analytics.NewProperties()

	// gather all flags (all flags must start with "flag_"
	// TODO: does it make sense to have a column for each flag, breaking if a new flag gets added?
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

	// set additional properties
	properties.Set("os_name", runtime.GOOS)
	properties.Set("os_arch", runtime.GOARCH)
	properties.Set("total_duration", time.Since(startTime).Milliseconds())

	if runtimeError == nil {
		properties.Set("runtime_error", "")
	} else {
		properties.Set("runtime_error", runtimeError.Error())
	}

	// TODO: add more properties?

	return properties
}

func Send(event string, startTime time.Time, runtimeError error) {
	// if the user opts out we return immediately
	if flags.OptOut {
		utils.Logger.Debug().Msg("opting-out of metric collection")
		return
	}

	userId, err := getUserId()
	if err != nil {
		utils.Logger.Debug().Err(err).Msg("failed to get user id")
	}

	// TODO: create new Segment key
	//client := analytics.New("")
	//defer client.Close()

	message := analytics.Track{
		UserId:     userId,
		Event:      event,
		Context:    getContext(),
		Properties: getProperties(startTime, runtimeError),
	}

	//if err := client.Enqueue(message); err != nil {
	//	utils.Logger.Debug().Err(err).Msg("failed to enqueue track message")
	//	return
	//}

	utils.Logger.Debug().Str("event", message.Event).Str("userId", message.UserId).Any("context", message.Context).Any("properties", message.Properties).Msg("sent track message")
}
