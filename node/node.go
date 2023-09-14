package node

import (
	"encoding/json"
	"fmt"
	cfg "github.com/KYVENetwork/ksync/config"
	"github.com/KYVENetwork/ksync/executors/blocksync/db/store"
	log "github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/node/helpers"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	logger = log.Logger("node")
)

type Node struct {
	DaemonPath string
	HomePath   string
	Mode       string
	PId        int
	Seeds      string
}

func NewNode(daemonPath string, homePath string, seeds string, mode string) *Node {
	if mode == "p2p" {
		err := helpers.SetConfig(homePath, true)
		if err != nil {
			return nil
		}
		logger.Info().Msg("successfully set up config")
	}
	return &Node{DaemonPath: daemonPath, HomePath: homePath, Mode: mode, PId: -1, Seeds: seeds}
}

func (n *Node) Start(flags string) error {
	var args []string

	if strings.HasSuffix(n.DaemonPath, "/cosmovisor") {
		args = []string{
			"run",
			"start",
		}
	} else {
		args = []string{
			"start",
		}
	}

	if n.Mode == "p2p" {
		// Move address book if exists.
		addrBookPath := filepath.Join(n.HomePath, "config", "addrbook.json")
		_, err := os.Stat(addrBookPath)
		if !os.IsNotExist(err) {
			if err = helpers.MoveFile(filepath.Join(n.HomePath, "config"), filepath.Join(n.HomePath), "addrbook.json"); err != nil {
				return err
			}
		}
	} else if n.Mode == "db" {
		args = append(args, "--with-tendermint=false")
	}

	args = append(args, "--home", n.HomePath, "--x-crisis-skip-assert-invariants", flags)

	if n.Mode == "normal" {
		if n.Seeds != "" {
			args = append(args, "--p2p-seeds", n.Seeds)
		} else {
			logger.Info().Msg("could not find seeds to connect")
		}
	}

	cmdPath, err := exec.LookPath(n.DaemonPath)
	if err != nil {
		return fmt.Errorf("could not resolve binary path: %s", err)
	}

	cmd := exec.Command(cmdPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	processIDChan := make(chan int)

	go func() {
		err = cmd.Start()
		if err != nil {
			logger.Error().Str("could not start node process", err.Error())
			processIDChan <- -1
			return
		}

		processIDChan <- cmd.Process.Pid

		// Wait for process end
		err = cmd.Wait()
		if err != nil {
			// Process can only be stopped through an error, not necessary to log an error
			processIDChan <- -1
		}
	}()

	processID := <-processIDChan

	if processID == -1 {
		return fmt.Errorf("could not start running the node")
	}

	_, err = os.FindProcess(processID)
	if err != nil {
		return fmt.Errorf("could not find started process: %s", err)
	}

	n.PId = processID

	return nil
}

func (n *Node) ShutdownNode(p2p bool) error {
	if p2p {
		err := helpers.SetConfig(n.HomePath, p2p)
		if err != nil {
			return err
		}
	} else {
		err := helpers.SetConfig(n.HomePath, p2p)
		if err != nil {
			return err
		}
	}
	if n.PId != -1 {
		process, err := os.FindProcess(n.PId)
		if err != nil {
			return fmt.Errorf("could not find process to shutdown: %s", err)
		}

		if err = process.Signal(syscall.SIGTERM); err != nil {
			return fmt.Errorf("could not terminate process: %s", err)
		}

		time.Sleep(time.Second * 30)

		n.PId = -1
	}

	return nil
}

// The GetNodeHeightDB function retrieves the height of the node by querying the BlockstoreDB.
func GetNodeHeightDB(home string) (int64, error) {
	config, err := cfg.LoadConfig(home)
	if err != nil {
		return 0, err
	}

	blockStoreDB, blockStore, err := store.GetBlockstoreDBs(config)
	defer blockStoreDB.Close()

	if err != nil {
		return 0, err
	}

	height := blockStore.Height()
	return height, nil
}

// The GetNodeHeightURL function retrieves the height of the node by querying the ABCI endpoint.
// It uses recursion with a maximum depth of 10 to handle delays or failures.
// It returns the nodeHeight if successful or an error message if the recursion depth reaches the limit (200s).
func (n *Node) GetNodeHeightURL(recursionDepth int) (int64, error) {
	response, err := http.Get(utils.ABCIEndpoint)
	if recursionDepth < 30 {
		if err != nil {
			logger.Info().Msg(fmt.Sprintf("could not query node height. Try again in 20s ... (%d/30)", recursionDepth+1))

			time.Sleep(time.Second * 20)
			return n.GetNodeHeightURL(recursionDepth + 1)
		} else {
			responseData, err := io.ReadAll(response.Body)
			if err != nil {
				logger.Error().Str("could not read response data", err.Error())
			}

			var resp types.HeightResponse
			err = json.Unmarshal(responseData, &resp)
			if err != nil {
				logger.Error().Str("could not unmarshal JSON", err.Error())
			}

			lastBlockHeight := resp.Result.Response.LastBlockHeight
			var nodeHeight int64

			if lastBlockHeight != "" {
				nodeHeight, err = strconv.ParseInt(lastBlockHeight, 10, 64)
				if err != nil {
					logger.Error().Str("could not convert lastBlockHeight to int; set it to 0", err.Error())
					nodeHeight = 0
				}
			} else {
				logger.Error().Msg("lastBlockHeight is empty; set it to 0")
				nodeHeight = 0
			}

			return nodeHeight, nil
		}
	} else {
		return 0, fmt.Errorf("could not get node height, exiting")
	}
}
