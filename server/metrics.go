package server

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/tendermint/tendermint/libs/json"
	"github.com/tendermint/tendermint/store"
	"net/http"
	"time"
)

type MetricsServer struct {
	blockStore *store.BlockStore
	port       int64
}

func StartMetricsApiServer(blockStore *store.BlockStore, port int64) *MetricsServer {
	metricsServer := &MetricsServer{
		blockStore: blockStore,
		port:       port,
	}

	r := gin.New()

	r.GET("/metrics", metricsServer.MetricsHandler)

	if err := r.Run(fmt.Sprintf(":%d", port)); err != nil {
		panic(err)
	}

	return metricsServer
}

func (metricsServer *MetricsServer) MetricsHandler(c *gin.Context) {
	type Metrics struct {
		LatestBlockHash     string    `json:"latest_block_hash"`
		LatestAppHash       string    `json:"latest_app_hash"`
		LatestBlockHeight   int64     `json:"latest_block_height"`
		LatestBlockTime     time.Time `json:"latest_block_time"`
		EarliestBlockHash   string    `json:"earliest_block_hash"`
		EarliestAppHash     string    `json:"earliest_app_hash"`
		EarliestBlockHeight int64     `json:"earliest_block_height"`
		EarliestBlockTime   time.Time `json:"earliest_block_time"`
		CatchingUp          bool      `json:"catching_up"`
	}

	latest := metricsServer.blockStore.LoadBlock(metricsServer.blockStore.Height())
	earliest := metricsServer.blockStore.LoadBlock(metricsServer.blockStore.Base())

	resp, err := json.Marshal(Metrics{
		LatestBlockHash:     latest.Header.Hash().String(),
		LatestAppHash:       latest.AppHash.String(),
		LatestBlockHeight:   latest.Height,
		LatestBlockTime:     latest.Time,
		EarliestBlockHash:   earliest.Hash().String(),
		EarliestAppHash:     earliest.AppHash.String(),
		EarliestBlockHeight: earliest.Height,
		EarliestBlockTime:   earliest.Time,
		CatchingUp:          true,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Error marshalling response"),
		})
		return
	}

	c.Data(http.StatusOK, "application/json", resp)
}
