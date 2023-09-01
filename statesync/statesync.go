package statesync

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/statesync/helpers"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	abciClient "github.com/tendermint/tendermint/abci/client"
	abci "github.com/tendermint/tendermint/abci/types"
	tmCfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/proxy"
	"net/http"
	"os"
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

	var chunkIds = []string{
		"8995b2e7-dabd-4564-9cc5-f54f4400c5ee",
		"5b57784e-3425-444a-9e6b-396d56d3b38b",
	}

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

	trustedAppHash, err := hex.DecodeString(bundle0[0].Value.AppHash)
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

	os.Exit(0)

	// TODO: Fetch closest snapshot containing all chunks with the specified height
	targetSnapshot := Snapshot{
		Height:         100,
		Format:         1,
		Chunks:         2,
		Hash:           []byte("z099DkY3lUWxFq+fVFQZQUC1SBsCQEERoSbufBOE9E8="),
		Metadata:       nil,
		TrustedAppHash: []byte("43694459343667617334493763655A76425258766D6A4F4B5245584376633169774E2B2B515A54547345626432516F6734536C6D78453471493032534F4C3464366C65653869716F48494F2B76753330336E766F4C465A6B54706B3D"),
	}

	var chunks []Chunk

	var i uint32
	for i = 0; i < targetSnapshot.Chunks; i++ {
		res, err := http.Get(fmt.Sprintf("https://storage.kyve.network/%v", chunkIds[i]))
		if err != nil {
			return err
		}
		var chunk = Chunk{
			Height: targetSnapshot.Height,
			Format: targetSnapshot.Format,
			Index:  i,
			Chunk:  nil,
			Sender: "",
		}

		var items []Item

		decoder := json.NewDecoder(res.Body)
		if err := decoder.Decode(&items); err != nil {
			panic("Error decoding JSON")
		}

		if len(items) > 0 {
			chunk.Chunk = []byte(items[0].Chunk)
		} else {
			fmt.Println("No items found in the JSON array")
		}

		if err != nil {
			return err
		}
		chunks = append(chunks, chunk)
	}

	accept, err := offerSnapshot(targetSnapshot)
	if err != nil {
		return err
	}
	if !accept {
		return fmt.Errorf("node did not accept snapshot offer")
	}

	err = applyChunks(chunks)
	if err != nil {
		return err
	}

	appVersion, err := helpers.GetAppVersion(config)
	if err != nil {
		return err
	}

	err = verifyApp(config, &targetSnapshot, appVersion)
	if err != nil {
		return err
	}

	logger.Info().Msg("successfully synced")

	return nil
}

func offerSnapshot(snapshot Snapshot) (accept bool, err error) {
	socketClient := abciClient.NewSocketClient("tcp://0.0.0.0:26658", false)

	if err := socketClient.Start(); err != nil {
		return false, err
	}

	logger.Info().Bytes("snapshot metadata", snapshot.Metadata).Msg("starting offering")

	res, err := socketClient.OfferSnapshotSync(abci.RequestOfferSnapshot{
		Snapshot: &abci.Snapshot{
			Height:   snapshot.Height,
			Format:   snapshot.Format,
			Chunks:   snapshot.Chunks,
			Hash:     snapshot.Hash,
			Metadata: snapshot.Metadata,
		},
		AppHash: snapshot.TrustedAppHash,
	})

	if err := socketClient.Stop(); err != nil {
		return false, err
	}

	switch res.Result {
	case abci.ResponseOfferSnapshot_ACCEPT:
		return true, nil
	case abci.ResponseOfferSnapshot_ABORT:
		return false, errAbort
	case abci.ResponseOfferSnapshot_REJECT:
		return false, errRejectSnapshot
	case abci.ResponseOfferSnapshot_REJECT_FORMAT:
		return false, errRejectFormat
	case abci.ResponseOfferSnapshot_REJECT_SENDER:
		return false, errRejectSender
	default:
		return false, fmt.Errorf("unknown ResponseOfferSnapshot result %v", res.Result)
	}
}

// applyChunks applies chunks to the app. It returns various errors depending on the app's
// response, or nil once the snapshot is fully restored.
func applyChunks(chunks []Chunk) error {
	socketClient := abciClient.NewSocketClient("tcp://0.0.0.0:26658", false)

	logger.Info().Msg("created socket client successfully")
	if err := socketClient.Start(); err != nil {
		logger.Error().Str("could not start socketClient", err.Error())
		return err
	}

	logger.Info().Msg("started socket client successfully")

	for _, chunk := range chunks {
		resp, err := socketClient.ApplySnapshotChunkSync(abci.RequestApplySnapshotChunk{
			Index:  chunk.Index,
			Chunk:  chunk.Chunk,
			Sender: string(chunk.Sender),
		})
		if err != nil {
			return fmt.Errorf("failed to apply chunk %v: %w", chunk.Index, err)
		}
		logger.Info().Uint64("height", chunk.Height).Uint32("format", chunk.Format).Uint32("chunk", chunk.Index).Msg("Applied snapshot chunk to ABCI app")

		switch resp.Result {
		case abci.ResponseApplySnapshotChunk_ACCEPT:
		case abci.ResponseApplySnapshotChunk_ABORT:
			return errAbort
		case abci.ResponseApplySnapshotChunk_RETRY:
			// TODO(@christopher): Rerun this specific loop period.
			return fmt.Errorf("retry is not implemented")
		case abci.ResponseApplySnapshotChunk_RETRY_SNAPSHOT:
			return errRetrySnapshot
		case abci.ResponseApplySnapshotChunk_REJECT_SNAPSHOT:
			return errRejectSnapshot
		default:
			return fmt.Errorf("unknown ResponseApplySnapshotChunk result %v", resp.Result)
		}
	}
	return nil
}

// verifyApp verifies the sync, checking the app hash, last block height and app version
func verifyApp(config *tmCfg.Config, snapshot *Snapshot, appVersion uint64) error {
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

	// sanity check that the app version in the block matches the application's own record
	// of its version
	if resp.AppVersion != appVersion {
		// An error here most likely means that the app hasn't implemented state sync
		// or the Info call correctly
		return fmt.Errorf("app version mismatch. Expected: %d, got: %d",
			appVersion, resp.AppVersion)
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
