package statesync

import (
	"fmt"
	"github.com/KYVENetwork/ksync/binary"
	"github.com/KYVENetwork/ksync/types"
)

// StartStateSyncExecutor takes the bundle id of the first snapshot chunk and applies the snapshot from there
func StartStateSyncExecutor(app *binary.CosmosApp, snapshotCollector types.SnapshotCollector, snapshotHeight int64) error {
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

	snapshot, err := snapshotCollector.GetSnapshotFromBundleId(bundleId)
	if err != nil {
		return fmt.Errorf("failed to get snapshot from bundle id %d: %w", bundleId, err)
	}

	snapshotHeight, chunks, err := app.ConsensusEngine.OfferSnapshot(snapshot.Value.Snapshot, snapshot.Value.State)
	if err != nil {
		return fmt.Errorf("failed to offer snapshot: %w", err)
	}

	logger.Info().Msgf("offering snapshot for height %d: ACCEPT", snapshotHeight)

	if err := app.ConsensusEngine.ApplySnapshotChunk(0, snapshot.Value.Chunk); err != nil {
		return fmt.Errorf("applying snapshot chunk %d/%d failed: %w", 1, chunks, err)
	}

	logger.Info().Msgf("applied snapshot chunk %d/%d: ACCEPT", 1, chunks)

	for chunkIndex := int64(1); chunkIndex < chunks; chunkIndex++ {
		chunk, err := snapshotCollector.DownloadChunkFromBundleId(bundleId + chunkIndex)
		if err != nil {
			return fmt.Errorf("failed downloading snapshot chunk from bundle id %d: %w", bundleId+chunkIndex, err)
		}

		logger.Info().Msgf("downloaded snapshot chunk %d/%d", chunkIndex+1, chunks)

		if err := app.ConsensusEngine.ApplySnapshotChunk(chunkIndex, chunk); err != nil {
			return fmt.Errorf("applying snapshot chunk %d/%d failed: %w", chunkIndex+1, chunks, err)
		}

		logger.Info().Msgf("applied snapshot chunk %d/%d: ACCEPT", chunkIndex+1, chunks)
	}

	if err := app.ConsensusEngine.BootstrapState(snapshot.Value.State, snapshot.Value.SeenCommit, snapshot.Value.Block); err != nil {
		return fmt.Errorf("failed to bootstrap state after state-sync: %w", err)
	}

	return nil
}
