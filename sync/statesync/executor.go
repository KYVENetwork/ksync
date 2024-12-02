package statesync

import (
	"fmt"
	"github.com/KYVENetwork/ksync/app"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	"github.com/tendermint/tendermint/libs/json"
)

// StartStateSyncExecutor takes the bundle id of the first snapshot chunk and applies the snapshot from there
func StartStateSyncExecutor(app *app.CosmosApp, snapshotCollector types.SnapshotCollector, snapshotHeight int64) error {
	if snapshotCollector == nil {
		return fmt.Errorf("snapshot collector can't be nil")
	}

	appHeight, err := app.ConsensusEngine.GetAppHeight()
	if err != nil {
		return fmt.Errorf("failed to get height from cosmos app: %w", err)
	}

	if appHeight > 0 {
		return fmt.Errorf("app height %d is not zero, please reset with \"ksync reset-all\" or run the command with \"--reset-all\"", appHeight)
	}

	bundleId, err := snapshotCollector.FindSnapshotBundleIdForHeight(snapshotHeight)
	if err != nil {
		return fmt.Errorf("failed to find snapshot bundle id for height %d: %w", snapshotHeight, err)
	}

	snapshotDataItem, err := snapshotCollector.GetSnapshotFromBundleId(bundleId)
	if err != nil {
		return fmt.Errorf("failed to get snapshot from bundle id %d: %w", bundleId, err)
	}

	var snapshot types.Snapshot
	if err := json.Unmarshal(snapshotDataItem.Value.Snapshot, &snapshot); err != nil {
		return fmt.Errorf("failed to unmarshal snapshot from bundle id %d: %w", bundleId, err)
	}

	if err := app.ConsensusEngine.OfferSnapshot(snapshotDataItem.Value.Snapshot, snapshotDataItem.Value.State); err != nil {
		return fmt.Errorf("failed to offer snapshot: %w", err)
	}

	utils.Logger.Info().Msgf("offering snapshot for height %d: ACCEPT", snapshot.Height)

	if err := app.ConsensusEngine.ApplySnapshotChunk(0, snapshotDataItem.Value.Chunk); err != nil {
		return fmt.Errorf("applying snapshot chunk %d/%d failed: %w", 1, snapshot.Chunks, err)
	}

	utils.Logger.Info().Msgf("applied snapshot chunk %d/%d: ACCEPT", 1, snapshot.Chunks)

	for chunkIndex := int64(1); chunkIndex < int64(snapshot.Chunks); chunkIndex++ {
		chunk, err := snapshotCollector.DownloadChunkFromBundleId(bundleId + chunkIndex)
		if err != nil {
			return fmt.Errorf("failed downloading snapshot chunk from bundle id %d: %w", bundleId+chunkIndex, err)
		}

		utils.Logger.Info().Msgf("downloaded snapshot chunk %d/%d", chunkIndex+1, snapshot.Chunks)

		if err := app.ConsensusEngine.ApplySnapshotChunk(chunkIndex, chunk); err != nil {
			return fmt.Errorf("applying snapshot chunk %d/%d failed: %w", chunkIndex+1, snapshot.Chunks, err)
		}

		utils.Logger.Info().Msgf("applied snapshot chunk %d/%d: ACCEPT", chunkIndex+1, snapshot.Chunks)
	}

	if err := app.ConsensusEngine.BootstrapState(snapshotDataItem.Value.State, snapshotDataItem.Value.SeenCommit, snapshotDataItem.Value.Block); err != nil {
		return fmt.Errorf("failed to bootstrap state after state-sync: %w", err)
	}

	utils.Logger.Info().Uint64("height", snapshot.Height).Uint32("format", snapshot.Format).Str("hash", fmt.Sprintf("%X", snapshot.Hash)).Msg("snapshot restored")
	return nil
}
