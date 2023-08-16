package server

import (
	"fmt"
	"github.com/gin-gonic/gin"
	abciClient "github.com/tendermint/tendermint/abci/client"
	"github.com/tendermint/tendermint/abci/types"
	"net/http"
	"strconv"
)

func StartApiServer() {
	r := gin.New()

	r.GET("/list_snapshots", ListSnapshotsHandler)
	r.GET("/load_snapshot_chunk/:height/:format/:chunk", LoadSnapshotChunkHandler)

	if err := r.Run(":7878"); err != nil {
		panic(err)
	}
}

func ListSnapshotsHandler(c *gin.Context) {
	// TODO: use config.ProxyApp for socket connection
	socketClient := abciClient.NewSocketClient("tcp://0.0.0.0:26658", false)

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

func LoadSnapshotChunkHandler(c *gin.Context) {
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

	socketClient := abciClient.NewSocketClient("tcp://0.0.0.0:26658", false)

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
