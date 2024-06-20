package server

import (
	"encoding/json"
	"fmt"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

var logger = utils.KsyncLogger("server")

type BlockApiServer struct {
	engine types.Engine
	port   int64
}

func StartBlockApiServer(engine types.Engine, port int64) *BlockApiServer {
	apiServer := &BlockApiServer{
		engine: engine,
		port:   port,
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	r.GET("/block", apiServer.GetBlockHandler)
	r.GET("/block_results", apiServer.GetBlockResultsHandler)

	logger.Info().Msgf("Starting block API server on port %d", port)
	if err := r.Run(fmt.Sprintf(":%d", port)); err != nil {
		panic(err)
	}

	return apiServer
}

func (bs *BlockApiServer) GetBlockHandler(c *gin.Context) {
	height, err := bs.getHeightFromQueryParamOrLatest(c)
	if err != nil {
		returnJsonRpcError(c, -32600, "error parsing height", err.Error())
		return
	}
	block, err := bs.engine.GetBlockWithMeta(height)
	if err != nil {
		returnJsonRpcError(c, -32600, "error getting block", err.Error())
		return
	}

	resp, err := toJsonRpcResponse(block)
	if err != nil {
		returnJsonRpcError(c, -32600, "error marshalling response", err.Error())
		return
	}

	c.Data(http.StatusOK, "application/json", resp)
}

func (bs *BlockApiServer) GetBlockResultsHandler(c *gin.Context) {
	height, err := bs.getHeightFromQueryParamOrLatest(c)
	if err != nil {
		returnJsonRpcError(c, -32600, "error parsing height", err.Error())
		return
	}
	blockResults, err := bs.engine.GetBlockResults(height)
	if err != nil {
		returnJsonRpcError(c, -32600, "error getting block results", err.Error())
		return
	}

	resp, err := toJsonRpcResponse(blockResults)
	if err != nil {
		returnJsonRpcError(c, -32600, "error marshalling response", err.Error())
		return
	}

	c.Data(http.StatusOK, "application/json", resp)
}

func (bs *BlockApiServer) getHeightFromQueryParamOrLatest(c *gin.Context) (int64, error) {
	heightStr, ok := c.GetQuery("height")
	if ok {
		height, err := strconv.ParseInt(heightStr, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("error parsing param \"height\" to int64: %s", err.Error())
		}
		return height, nil
	}
	return bs.engine.GetHeight(), nil
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

type rpcResponse struct {
	Jsonrpc string          `json:"jsonrpc"`
	Id      int             `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

func toJsonRpcResponse(result []byte) ([]byte, error) {
	response := rpcResponse{
		Jsonrpc: "2.0",
		Id:      -1,
		Result:  result,
	}

	return json.Marshal(response)
}

func toJsonRpcError(code int, message string, data string) ([]byte, error) {
	response := rpcResponse{
		Jsonrpc: "2.0",
		Id:      -1,
		Error: &rpcError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}

	return json.Marshal(response)
}

func returnJsonRpcError(c *gin.Context, code int, message string, data string) {
	rpcErr, err := toJsonRpcError(code, message, data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.Data(http.StatusInternalServerError, "application/json", rpcErr)
}
