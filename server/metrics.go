package server

import (
	"fmt"
	"github.com/KYVENetwork/ksync/types"
	"github.com/gin-gonic/gin"
	"net/http"
)

type MetricsServer struct {
	engine types.Engine
	port   int64
}

func StartMetricsApiServer(engine types.Engine, port int64) *MetricsServer {
	metricsServer := &MetricsServer{
		engine: engine,
		port:   port,
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	r.GET("/metrics", metricsServer.MetricsHandler)

	if err := r.Run(fmt.Sprintf(":%d", port)); err != nil {
		panic(err)
	}

	return metricsServer
}

func (metricsServer *MetricsServer) MetricsHandler(c *gin.Context) {
	metrics, err := metricsServer.engine.GetMetrics()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Error marshalling response"),
		})
		return
	}

	c.Data(http.StatusOK, "application/json", metrics)
}
