package statesync

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	abciClient "github.com/tendermint/tendermint/abci/client"
	abci "github.com/tendermint/tendermint/abci/types"
	tmCfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/proxy"
)

type Item struct {
	Key   string `json:"key"`
	Value Value  `json:"value"`
	Index int    `json:"index"`
	Chunk string `json:"chunk"`
}

type Value struct {
	Snapshot SnapshotRes `json:"snapshot"`
	Index    int         `json:"index"`
	Chunk    string      `json:"chunk"`
}

type SnapshotRes struct {
	Height   int    `json:"height"`
	Format   int    `json:"format"`
	Chunks   int    `json:"chunks"`
	Hash     string `json:"hash"`
	Metadata string `json:"metadata"`
}

// Snapshot contains data about a snapshot.
type Snapshot struct {
	Height   uint64
	Format   uint32
	Chunks   uint32
	Hash     []byte
	Metadata []byte

	TrustedAppHash []byte // populated by light client
}

var (
	logger = log.Logger("state-sync")
	// errAbort is returned by Sync() when snapshot restoration is aborted.
	errAbort = errors.New("state sync aborted")
	// errRejectSnapshot is returned by Sync() when the snapshot is rejected.
	errRejectSnapshot = errors.New("snapshot was rejected")
	// errRejectFormat is returned by Sync() when the snapshot format is rejected.
	errRejectFormat = errors.New("snapshot format was rejected")
	// errRejectSender is returned by Sync() when the snapshot sender is rejected.
	errRejectSender = errors.New("snapshot sender was rejected")
	// errRetrySnapshot is returned by Sync() when the snapshot should be retried.
	errRetrySnapshot = errors.New("retry snapshot")
	// errVerifyFailed is returned by Sync() when app hash or last height verification fails.
	errVerifyFailed = errors.New("verification failed")
)

func StartStateSync(config *tmCfg.Config) error {
	logger.Info().Msg("starting state-sync")

	chunk0Raw, chunk0Err := utils.DownloadFromUrl("https://storage.kyve.network/f95d2fe1-0c6a-4a68-9ea0-624127df4be7")
	if chunk0Err != nil {
		panic(fmt.Errorf("error downloading chunk 0 %w", chunk0Err))
	}
	chunk1Raw, chunk1Err := utils.DownloadFromUrl("https://storage.kyve.network/82b63fc9-896c-44f7-a192-c3fa5e32a48a")
	if chunk1Err != nil {
		panic(fmt.Errorf("error downloading chunk 1 %w", chunk1Err))
	}

	chunk0, chunk0Err := utils.DecompressGzip(chunk0Raw)
	if chunk0Err != nil {
		panic(fmt.Errorf("error decompressing chunk 0 %w", chunk0Err))
	}

	chunk1, chunk1Err := utils.DecompressGzip(chunk1Raw)
	if chunk1Err != nil {
		panic(fmt.Errorf("error decompressing chunk 1 %w", chunk1Err))
	}

	var bundle0 types.TendermintSsyncBundle
	var bundle1 types.TendermintSsyncBundle

	if err := json.Unmarshal(chunk0, &bundle0); err != nil {
		panic(fmt.Errorf("failed to unmarshal tendermint bundle: %w", err))
	}

	if err := json.Unmarshal(chunk1, &bundle1); err != nil {
		panic(fmt.Errorf("failed to unmarshal tendermint bundle: %w", err))
	}

	fmt.Println(bundle0[0].Key)
	fmt.Println(bundle0[0].Value.AppHash)
	fmt.Println([]byte(bundle0[0].Value.AppHash))
	fmt.Println(hex.DecodeString(bundle0[0].Value.AppHash))

	socketClient := abciClient.NewSocketClient("tcp://0.0.0.0:26658", false)

	if err := socketClient.Start(); err != nil {
		panic(fmt.Errorf("error starting abci server %w", err))
	}

	trustedAppHash, err := hex.DecodeString("E312C88409FD62DAB5BBACA4573483BF8DD676416C19FF63629CA069FD2D00AC")
	if err != nil {
		panic(fmt.Errorf("error decoding app hash %w", err))
	}

	res, err := socketClient.OfferSnapshotSync(abci.RequestOfferSnapshot{
		Snapshot: &abci.Snapshot{
			Height:   bundle0[0].Value.Snapshot.Height,
			Format:   bundle0[0].Value.Snapshot.Format,
			Chunks:   bundle0[0].Value.Snapshot.Chunks,
			Hash:     bundle0[0].Value.Snapshot.Hash,
			Metadata: bundle0[0].Value.Snapshot.Metadata,
		},
		AppHash: trustedAppHash,
	})

	fmt.Println(res)

	nodeKey, err := p2p.LoadNodeKey(config.NodeKeyFile())

	resp0, err := socketClient.ApplySnapshotChunkSync(abci.RequestApplySnapshotChunk{
		Index:  bundle0[0].Value.ChunkIndex,
		Chunk:  bundle0[0].Value.Chunk,
		Sender: string(nodeKey.ID()),
	})

	fmt.Println(resp0)

	resp1, err := socketClient.ApplySnapshotChunkSync(abci.RequestApplySnapshotChunk{
		Index:  bundle1[0].Value.ChunkIndex,
		Chunk:  bundle1[0].Value.Chunk,
		Sender: string(nodeKey.ID()),
	})

	fmt.Println(resp1)

	testSnapshot := Snapshot{
		Height:         bundle0[0].Value.Snapshot.Height,
		Format:         bundle0[0].Value.Snapshot.Format,
		Chunks:         bundle0[0].Value.Snapshot.Chunks,
		Hash:           bundle0[0].Value.Snapshot.Hash,
		Metadata:       bundle0[0].Value.Snapshot.Metadata,
		TrustedAppHash: trustedAppHash,
	}

	if err != verifyApp(config, &testSnapshot) {
		panic(fmt.Errorf("error verifying app %w", err))
	}

	return nil
}

// verifyApp verifies the sync, checking the app hash, last block height and app version
func verifyApp(config *tmCfg.Config, snapshot *Snapshot) error {
	socketClient := abciClient.NewSocketClient(config.ProxyApp, false)

	logger.Info().Msg("created socket client successfully")
	if err := socketClient.Start(); err != nil {
		logger.Error().Str("could not start socketClient", err.Error())
		return err
	}

	resp, err := socketClient.InfoSync(proxy.RequestInfo)
	if err != nil {
		return fmt.Errorf("failed to query ABCI app for appHash: %w", err)
	}

	if !bytes.Equal(snapshot.TrustedAppHash, resp.LastBlockAppHash) {
		logger.Error().Bytes("expected", snapshot.TrustedAppHash).Bytes("actual", resp.LastBlockAppHash).Msg("appHash verification failed")
		return errVerifyFailed
	}
	if uint64(resp.LastBlockHeight) != snapshot.Height {
		logger.Error().Uint64("expected", snapshot.Height).Int64("actual", resp.LastBlockHeight).Msg("ABCI app reported unexpected last block height")
		return errVerifyFailed
	}

	logger.Info().Bytes("appHash", snapshot.TrustedAppHash).Uint64("height", snapshot.Height).Msg("Verified ABCI app")
	return nil
}
