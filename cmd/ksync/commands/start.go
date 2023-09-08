package commands

import (
	"fmt"
	"github.com/KYVENetwork/ksync/backup/helpers"
	"github.com/KYVENetwork/ksync/config"
	"github.com/KYVENetwork/ksync/executor/auto"
	"github.com/KYVENetwork/ksync/executor/db"
	"github.com/KYVENetwork/ksync/executor/p2p"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/spf13/cobra"
	"path/filepath"
	"strings"
)

var (
	apiServer    bool
	backupCompression string
	backupDest        string
	backupInterval    int
	backupKeepRecent  int
	chainId           string
	daemonPath        string
	flags             string
	mode              string
	home              string
	poolId            int64
	port         int64
	restEndpoint      string
	seeds             string
	targetHeight      int64

	quitCh = make(chan int)
)

func init() {
	startCmd.Flags().StringVar(&mode, "mode", utils.DefaultMode, fmt.Sprintf("sync mode (\"auto\",\"db\",\"p2p\"), [default = %s]", utils.DefaultMode))

	startCmd.Flags().StringVar(&home, "home", "", "home directory")
	if err := startCmd.MarkFlagRequired("home"); err != nil {
		panic(fmt.Errorf("flag 'home' should be required: %w", err))
	}

	// Optional AUTO-MODE flags.
	startCmd.Flags().StringVar(&daemonPath, "daemon-path", "", "daemon path of node to be synced")

	startCmd.Flags().StringVar(&chainId, "chain-id", utils.DefaultChainId, fmt.Sprintf("kyve chain id (\"kyve-1\",\"kaon-1\",\"korellia\"), [default = %s]", utils.DefaultChainId))

	startCmd.Flags().Int64Var(&poolId, "pool-id", 0, "pool id")
	if err := startCmd.MarkFlagRequired("pool-id"); err != nil {
		panic(fmt.Errorf("flag 'pool-id' should be required: %w", err))
	}

	startCmd.Flags().StringVar(&restEndpoint, "rest-endpoint", "", "Overwrite default rest endpoint from chain")

	startCmd.Flags().Int64Var(&targetHeight, "target-height", 0, "target height (including)")

	startCmd.Flags().BoolVar(&apiServer, "api-server", false, "start an api server on http://localhost:7878 for a TSP connection to the tendermint app")

	startCmd.Flags().Int64Var(&port, "port", 7878, "change the port of the api server, [default = 7878]")

	startCmd.Flags().StringVar(&seeds, "seeds", "", "P2P seeds to continue syncing process after KSYNC")

	startCmd.Flags().StringVar(&flags, "flags", "", "Flags for starting the node to be synced; excluding --home and --with-tendermint")

	startCmd.Flags().StringVar(&backupDest, "backup-dest-path", "", "destination path of the written backup (default '~/.ksync/backups)'")

	backupCmd.Flags().StringVar(&backupCompression, "backup-compression", "tar.gz", "compression type to compress backup directory ['tar.gz', 'zip', '']")

	backupCmd.Flags().IntVar(&backupKeepRecent, "backup-keep-recent", 2, "number of kept backups (set 0 to keep all)")

	backupCmd.Flags().IntVar(&backupInterval, "backup-interval", 0, "block interval to write backups (set 0 to disable backups)")

	rootCmd.AddCommand(startCmd)
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start fast syncing blocks with KSYNC",
	Run: func(cmd *cobra.Command, args []string) {
		// if no custom rest endpoint was given we take it from the chainId
		if restEndpoint == "" {
			switch chainId {
			case "kyve-1":
				restEndpoint = utils.RestEndpointMainnet
			case "kaon-1":
				restEndpoint = utils.RestEndpointKaon
			case "korellia":
				restEndpoint = utils.RestEndpointKorellia
			default:
				panic("flag --chain-id has to be either \"kyve-1\", \"kaon-1\" or \"korellia\"")
			}
		}

		// trim trailing slash
		restEndpoint = strings.TrimSuffix(restEndpoint, "/")

		var backup = &types.BackupConfig{
			Compression: backupCompression,
			Src:         "",
			Dest:        backupDest,
			Interval:    backupInterval,
			KeepRecent:  backupKeepRecent,
		}

		if backupInterval > 0 {
			backup.Src = filepath.Join(home, "data")

			if backup.Dest == "" {
				backupDir, err := config.GetBackupDir()
				if err != nil {
					logger.Error().Str("err", err.Error()).Msg("failed to get ksync home directory")
					return
				}

				d, err := helpers.CreateDestPath(backupDir)
				if err != nil {
					logger.Error().Str("err", err.Error()).Msg("failed to create backup destination")
					return
				}
				backup.Dest = d
			}
			if err := helpers.ValidatePaths(backup.Src, backup.Dest); err != nil {
				logger.Error().Str("err", err.Error()).Msg("path validation failed")
				return
			}
		}

		// start block executor based on sync mode
		switch mode {
		case "auto":
			if daemonPath == "" {
				panic("flag --daemon-path is required for mode \"auto\"")
			}
			auto.StartAutoExecutor(quitCh, home, daemonPath, seeds, flags, poolId, restEndpoint, targetHeight, apiServer, port, backup)
		case "db":
			if apiServer {
				panic("flag --api-server not supported for mode \"db\"")
			}
			go db.StartDBExecutor(quitCh, home, poolId, restEndpoint, targetHeight, false, port)
		case "p2p":
			if apiServer {
				panic("flag --api-server not supported for mode \"p2p\"")
			}
			go p2p.StartP2PExecutor(quitCh, home, poolId, restEndpoint, targetHeight)
		default:
			panic("flag --mode has to be either \"auto\", \"db\" or \"p2p\"")
		}

		// only exit process if executor has finished
		<-quitCh
	},
}
