package statesync

import (
	"encoding/hex"
	"fmt"
	"github.com/KYVENetwork/ksync/executor/db/store"
	log "github.com/KYVENetwork/ksync/logger"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	abciClient "github.com/tendermint/tendermint/abci/client"
	abci "github.com/tendermint/tendermint/abci/types"
	tmCfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/json"
	"github.com/tendermint/tendermint/p2p"
	tmState "github.com/tendermint/tendermint/proto/tendermint/state"
	sm "github.com/tendermint/tendermint/state"
	tmTypes "github.com/tendermint/tendermint/types"
	"github.com/tendermint/tendermint/version"
)

var (
	logger = log.Logger("state-sync")
)

func StartStateSync(config *tmCfg.Config) error {
	logger.Info().Msg("starting state-sync")

	chunk0Raw, chunk0Err := utils.DownloadFromUrl("https://storage.kyve.network/393416aa-198a-434f-90f2-9cbdb0a97283")
	if chunk0Err != nil {
		panic(fmt.Errorf("error downloading chunk 0 %w", chunk0Err))
	}
	chunk1Raw, chunk1Err := utils.DownloadFromUrl("https://storage.kyve.network/4a3f39f0-e580-43fe-8bae-4bde8f8b249f")
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
		panic(fmt.Errorf("failed to unmarshal tendermint bundle 0: %w", err))
	}

	if err := json.Unmarshal(chunk1, &bundle1); err != nil {
		panic(fmt.Errorf("failed to unmarshal tendermint bundle 1: %w", err))
	}

	fmt.Println(bundle0[0].Key)
	fmt.Println(bundle0[0].Value.AppHash)
	fmt.Println([]byte(bundle0[0].Value.AppHash))
	fmt.Println(hex.DecodeString(bundle0[0].Value.AppHash))

	socketClient := abciClient.NewSocketClient(config.ProxyApp, false)

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

	state := sm.State{
		ChainID:         bundle0[0].Value.CurrentLightBlock.Header.ChainID,
		InitialHeight:   int64(bundle0[0].Value.Snapshot.Height),
		LastBlockHeight: bundle0[0].Value.LastLightBlock.Height,
		Version: tmState.Version{
			Consensus: bundle0[0].Value.CurrentLightBlock.Version,
			Software:  version.TMCoreSemVer,
		},
		LastBlockTime:                    bundle0[0].Value.LastLightBlock.Time,
		LastBlockID:                      bundle0[0].Value.CurrentLightBlock.Commit.BlockID,
		AppHash:                          bundle0[0].Value.CurrentLightBlock.AppHash,
		LastResultsHash:                  bundle0[0].Value.CurrentLightBlock.LastResultsHash,
		LastValidators:                   bundle0[0].Value.LastLightBlock.ValidatorSet,
		Validators:                       bundle0[0].Value.CurrentLightBlock.ValidatorSet,
		NextValidators:                   bundle0[0].Value.NextLightBlock.ValidatorSet,
		LastHeightValidatorsChanged:      bundle0[0].Value.NextLightBlock.Height,
		ConsensusParams:                  *bundle0[0].Value.ConsensusParams,
		LastHeightConsensusParamsChanged: bundle0[0].Value.CurrentLightBlock.Height,
	}

	if state.InitialHeight == 0 {
		state.InitialHeight = 1
	}

	stateDB, stateStore, err := store.GetStateDBs(config)
	defer stateDB.Close()

	err = stateStore.Bootstrap(state)
	if err != nil {
		panic(fmt.Errorf("failed to bootstrap node with new state: %w", err))
	}

	blockDB, blockStore, err := store.GetBlockstoreDBs(config)
	defer blockDB.Close()

	err = blockStore.SaveSeenCommit(state.LastBlockHeight, bundle0[0].Value.LastLightBlock.Commit)
	if err != nil {
		panic(fmt.Errorf("failed to store last seen commit: %w", err))
	}

	bundleRaw, bundleErr := utils.DownloadFromUrl("https://arweave.net/c-24-Ik7KGaB2WJyrW_2fsAjoJkaAD6xfk30qlqEpCI")
	if bundleErr != nil {
		panic(fmt.Errorf("error downloading bundle with blocks 0-150 %w", chunk0Err))
	}

	bundle, bundleErr := utils.DecompressGzip(bundleRaw)
	if bundleErr != nil {
		panic(fmt.Errorf("error decompressing bundlw with blocks 0-150 %w", chunk0Err))
	}

	var blocks types.TendermintBundle

	if err := json.Unmarshal(bundle, &blocks); err != nil {
		panic(fmt.Errorf("failed to unmarshal tendermint bundle with blocks 0-150: %w", err))
	}

	block100 := blocks[99].Value.Block.Block
	block101 := blocks[100].Value.Block.Block

	fmt.Println("Found block", block100.Height)

	blockParts := block100.MakePartSet(tmTypes.BlockPartSizeBytes)
	blockStore.SaveBlock(block100, blockParts, block101.LastCommit)

	return nil
}
