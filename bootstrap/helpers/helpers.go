package helpers

import (
	"fmt"
	"github.com/KYVENetwork/ksync/executors/blocksync/db/store"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/tendermint/tendermint/libs/json"
	"strconv"
	"strings"
)

func GetNodeHeightFromRPC(homePath string) (height int64, err error) {
	config, err := utils.LoadConfig(homePath)
	if err != nil {
		panic(fmt.Errorf("failed to load config.toml: %w", err))
	}

	rpc := fmt.Sprintf("%s/abci_info", strings.Replace(config.RPC.ListenAddress, "tcp", "http", 1))

	responseData, err := utils.GetFromUrl(rpc)
	if err != nil {
		return height, err
	}

	var response types.HeightResponse
	if err := json.Unmarshal(responseData, &response); err != nil {
		return height, err
	}

	if response.Result.Response.LastBlockHeight == "" {
		return 0, nil
	}

	height, err = strconv.ParseInt(response.Result.Response.LastBlockHeight, 10, 64)
	if err != nil {
		return height, err
	}

	return
}

func GetNodeHeightFromDB(home string) (int64, error) {
	config, err := utils.LoadConfig(home)
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
