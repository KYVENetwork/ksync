package server

import (
	"fmt"
	"github.com/KYVENetwork/ksync/types"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

type ApiServer struct {
	engine types.Engine
	port   int64
}

func StartSnapshotApiServer(engine types.Engine, port int64) *ApiServer {
	apiServer := &ApiServer{
		engine: engine,
		port:   port,
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	r.GET("/list_snapshots", apiServer.ListSnapshotsHandler)
	r.GET("/load_snapshot_chunk/:height/:format/:chunk", apiServer.LoadSnapshotChunkHandler)
	r.GET("/get_block/:height", apiServer.GetBlockHandler)
	r.GET("/get_state/:height", apiServer.GetStateHandler)
	r.GET("/get_seen_commit/:height", apiServer.GetSeenCommitHandler)

	if err := r.Run(fmt.Sprintf(":%d", port)); err != nil {
		panic(err)
	}

	return apiServer
}

func (apiServer *ApiServer) ListSnapshotsHandler(c *gin.Context) {
	resp, err := apiServer.engine.GetSnapshots()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Data(http.StatusOK, "application/json", resp)
}

func (apiServer *ApiServer) LoadSnapshotChunkHandler(c *gin.Context) {
	height, err := strconv.ParseInt(c.Param("height"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Error parsing param \"height\" to uint64: %s", err.Error()),
		})
		return
	}

	format, err := strconv.ParseInt(c.Param("format"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Error parsing param \"format\" to uint32: %s", err.Error()),
		})
		return
	}

	chunk, err := strconv.ParseInt(c.Param("chunk"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Error parsing param \"chunk\" to uint32: %s", err.Error()),
		})
		return
	}

	resp, err := apiServer.engine.GetSnapshotChunk(height, format, chunk)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
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

	resp, err := apiServer.engine.GetBlock(height)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
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

	resp, err := apiServer.engine.GetState(height)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Data(http.StatusOK, "application/json", resp)
}

func (apiServer *ApiServer) GetSeenCommitHandler(c *gin.Context) {
	height, err := strconv.ParseInt(c.Param("height"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Error parsing param \"height\" to uint64: %s", err.Error()),
		})
		return
	}

	resp, err := apiServer.engine.GetSeenCommit(height)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Data(http.StatusOK, "application/json", resp)
}
