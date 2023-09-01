package server

import (
	"fmt"
	"github.com/gin-gonic/gin"
	abciClient "github.com/tendermint/tendermint/abci/client"
	"github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/state"
	"github.com/tendermint/tendermint/store"
	tmTypes "github.com/tendermint/tendermint/types"
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
	r.GET("/get_app_hash/:height", apiServer.GetAppHashHandler)
	r.GET("/get_light_block/:height", apiServer.GetLightBlockHandler)

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

	c.JSON(http.StatusOK, res.Snapshots)
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

	c.JSON(http.StatusOK, res.Chunk)
}

func (apiServer *ApiServer) GetAppHashHandler(c *gin.Context) {
	height, err := strconv.ParseInt(c.Param("height"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Error parsing param \"height\" to uint64: %s", err.Error()),
		})
		return
	}

	type AppHash struct {
		AppHash string
	}

	block := apiServer.blockStore.LoadBlock(height)

	c.JSON(http.StatusOK, AppHash{
		AppHash: block.Header.AppHash.String(),
	})
}

func (apiServer *ApiServer) GetLightBlockHandler(c *gin.Context) {
	height, err := strconv.ParseInt(c.Param("height"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Error parsing param \"height\" to uint64: %s", err.Error()),
		})
		return
	}

	block := apiServer.blockStore.LoadBlock(height)

	validatorSet, err := apiServer.stateStore.LoadValidators(block.Height)

	lightBlock := tmTypes.LightBlock{
		SignedHeader: &tmTypes.SignedHeader{
			Header: &block.Header,
			Commit: block.LastCommit,
		},
		ValidatorSet: validatorSet,
	}

	c.JSON(http.StatusOK, lightBlock)
}
