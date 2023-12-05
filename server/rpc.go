package server

import (
	"fmt"
	"github.com/KYVENetwork/ksync/engines/tendermint"
	"github.com/KYVENetwork/ksync/types"
	abciClient "github.com/tendermint/tendermint/abci/client"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/bytes"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	rpcserver "github.com/tendermint/tendermint/rpc/jsonrpc/server"
	rpctypes "github.com/tendermint/tendermint/rpc/jsonrpc/types"
	"net/http"
	"strings"
	"time"
)

var (
	tmLogger = tendermint.TmLogger()
)

func ABCIQuery(
	_ *rpctypes.Context,
	path string,
	data bytes.HexBytes,
	height int64,
	prove bool,
) (*ctypes.ResultABCIQuery, error) {
	socketClient := abciClient.NewSocketClient("tcp://127.0.0.1:26658", false)

	if err := socketClient.Start(); err != nil {
		return nil, fmt.Errorf("failed to start socket client: %w", err)
	}

	resQuery, err := socketClient.QuerySync(abci.RequestQuery{
		Path:   path,
		Data:   data,
		Height: height,
		Prove:  prove,
	})
	if err != nil {
		return nil, err
	}

	return &ctypes.ResultABCIQuery{Response: *resQuery}, nil
}

var Routes = map[string]*rpcserver.RPCFunc{
	"abci_query": rpcserver.NewRPCFunc(ABCIQuery, "path,data,height,prove"),
}

func StartRPC(engine types.Engine) error {
	cfg, err := tendermint.LoadConfig(engine.GetHomePath())
	if err != nil {
		return fmt.Errorf("failed to load config.toml: %w", err)
	}

	config := rpcserver.DefaultConfig()
	config.MaxBodyBytes = cfg.RPC.MaxBodyBytes
	config.MaxHeaderBytes = cfg.RPC.MaxHeaderBytes
	config.MaxOpenConnections = cfg.RPC.MaxOpenConnections
	if config.WriteTimeout <= cfg.RPC.TimeoutBroadcastTxCommit {
		config.WriteTimeout = cfg.RPC.TimeoutBroadcastTxCommit + 1*time.Second
	}

	listenAddrs := splitAndTrimEmpty(cfg.RPC.ListenAddress, ",", " ")
	for _, listenAddr := range listenAddrs {
		fmt.Println("start rpc on ", listenAddr)
		mux := http.NewServeMux()

		rpcLogger := tmLogger.With("rpc")

		rpcserver.RegisterRPCFuncs(mux, Routes, rpcLogger)
		listener, err := rpcserver.Listen(
			listenAddr,
			config,
		)
		if err != nil {
			return fmt.Errorf("failed to listen: %w", err)
		}

		var rootHandler http.Handler = mux
		go func() {
			if err := rpcserver.Serve(
				listener,
				rootHandler,
				rpcLogger,
				config,
			); err != nil {
				fmt.Println(err.Error())
			}
		}()
	}

	return nil
}

func splitAndTrimEmpty(s, sep, cutset string) []string {
	if s == "" {
		return []string{}
	}

	spl := strings.Split(s, sep)
	nonEmptyStrings := make([]string, 0, len(spl))
	for i := 0; i < len(spl); i++ {
		element := strings.Trim(spl[i], cutset)
		if element != "" {
			nonEmptyStrings = append(nonEmptyStrings, element)
		}
	}
	return nonEmptyStrings
}
