package server

import (
	"fmt"
	"github.com/gin-gonic/gin"
	abciClient "github.com/tendermint/tendermint/abci/client"
	"github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/json"
	tmState "github.com/tendermint/tendermint/proto/tendermint/state"
	"github.com/tendermint/tendermint/state"
	"github.com/tendermint/tendermint/store"
	"github.com/tendermint/tendermint/version"
	"net/http"
	"strconv"
)

type ApiServer struct {
	config     *config.Config
	blockStore *store.BlockStore
	stateStore state.Store
	port       int64
}

func StartApiServer(config *config.Config, blockStore *store.BlockStore, stateStore state.Store, port int64) *ApiServer {
	apiServer := &ApiServer{
		config:     config,
		blockStore: blockStore,
		stateStore: stateStore,
	}

	r := gin.New()

	r.GET("/list_snapshots", apiServer.ListSnapshotsHandler)
	r.GET("/load_snapshot_chunk/:height/:format/:chunk", apiServer.LoadSnapshotChunkHandler)
	r.GET("/get_block/:height", apiServer.GetBlockHandler)
	r.GET("/get_state/:height", apiServer.GetStateHandler)

	if err := r.Run(fmt.Sprintf(":%d", port)); err != nil {
		panic(err)
	}

	return apiServer
}

func (apiServer *ApiServer) ListSnapshotsHandler(c *gin.Context) {
	socketClient := abciClient.NewSocketClient(apiServer.config.ProxyApp, false)

	if err := socketClient.Start(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	res, err := socketClient.ListSnapshotsSync(types.RequestListSnapshots{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err := socketClient.Stop(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	resp, err := json.Marshal(res.Snapshots)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Error marshalling response"),
		})
		return
	}

	c.Data(http.StatusOK, "application/json", resp)
}

func (apiServer *ApiServer) LoadSnapshotChunkHandler(c *gin.Context) {
	height, err := strconv.ParseUint(c.Param("height"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Error parsing param \"height\" to uint64: %s", err.Error()),
		})
		return
	}

	format, err := strconv.ParseUint(c.Param("format"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Error parsing param \"format\" to uint32: %s", err.Error()),
		})
		return
	}

	chunk, err := strconv.ParseUint(c.Param("chunk"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Error parsing param \"chunk\" to uint32: %s", err.Error()),
		})
		return
	}

	socketClient := abciClient.NewSocketClient(apiServer.config.ProxyApp, false)

	if err := socketClient.Start(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	res, err := socketClient.LoadSnapshotChunkSync(types.RequestLoadSnapshotChunk{
		Height: height,
		Format: uint32(format),
		Chunk:  uint32(chunk),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err := socketClient.Stop(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	resp, err := json.Marshal(res.Chunk)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Error marshalling response"),
		})
		return
	}

	c.Data(http.StatusOK, "application/json", resp)
}

func (apiServer *ApiServer) GetBlockHandler(c *gin.Context) {
	height, err := strconv.ParseInt(c.Param("height"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Error parsing param \"height\" to uint64: %s", err.Error()),
		})
		return
	}

	block := apiServer.blockStore.LoadBlock(height)

	resp, err := json.Marshal(block)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Error marshalling response"),
		})
		return
	}

	c.Data(http.StatusOK, "application/json", resp)
}

func (apiServer *ApiServer) GetStateHandler(c *gin.Context) {
	height, err := strconv.ParseInt(c.Param("height"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Error parsing param \"height\" to uint64: %s", err.Error()),
		})
		return
	}

	initialHeight := height
	if initialHeight == 0 {
		initialHeight = 1
	}

	lastBlock := apiServer.blockStore.LoadBlock(height)
	currentBlock := apiServer.blockStore.LoadBlock(height + 1)
	nextBlock := apiServer.blockStore.LoadBlock(height + 2)

	lastValidators, err := apiServer.stateStore.LoadValidators(height)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Error loading validator set at height %d: %s", height, err.Error()),
		})
		return
	}
	currentValidators, err := apiServer.stateStore.LoadValidators(height + 1)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Error loading validator set at height %d: %s", height+1, err.Error()),
		})
		return
	}
	nextValidators, err := apiServer.stateStore.LoadValidators(height + 2)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Error loading validator set at height %d: %s", height+2, err.Error()),
		})
		return
	}

	consensusParams, err := apiServer.stateStore.LoadConsensusParams(height)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Failed to load consensus params at height %d: %s", height, err.Error()),
		})
		return
	}

	snapshotState := state.State{
		Version: tmState.Version{
			Consensus: lastBlock.Version,
			Software:  version.TMCoreSemVer,
		},
		ChainID:                          lastBlock.ChainID,
		InitialHeight:                    initialHeight,
		LastBlockHeight:                  lastBlock.Height,
		LastBlockID:                      currentBlock.LastBlockID,
		LastBlockTime:                    lastBlock.Time,
		NextValidators:                   nextValidators,
		Validators:                       currentValidators,
		LastValidators:                   lastValidators,
		LastHeightValidatorsChanged:      nextBlock.Height,
		ConsensusParams:                  consensusParams,
		LastHeightConsensusParamsChanged: currentBlock.Height,
		LastResultsHash:                  currentBlock.LastResultsHash,
		AppHash:                          currentBlock.AppHash,
	}

	resp, err := json.Marshal(snapshotState)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Error marshalling response"),
		})
		return
	}

	c.Data(http.StatusOK, "application/json", resp)
}
