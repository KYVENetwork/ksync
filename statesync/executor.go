package statesync

import (
	"fmt"
	"github.com/KYVENetwork/ksync/collectors/bundles"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
)

// startStateSyncExecutor takes the bundle id of the first snapshot chunk and applies the snapshot from there
func startStateSyncExecutor(engine types.Engine, chainRest, storageRest string, snapshotPoolId, snapshotBundleId int64) error {
	logger.Info().Msg(fmt.Sprintf("applying state-sync snapshot"))

	appHeight, err := engine.GetAppHeight()
	if err != nil {
		return fmt.Errorf("requesting height from app failed: %w", err)
	}

	if appHeight > 0 {
		return fmt.Errorf("app height %d is not zero, please reset with \"ksync reset-all\" or run the command with \"--reset-all\"", appHeight)
	}

	finalizedBundle, err := bundles.GetFinalizedBundleById(chainRest, snapshotPoolId, snapshotBundleId)
	if err != nil {
		return fmt.Errorf("failed getting finalized bundle: %w", err)
	}

	snapshotHeight, _, err := utils.ParseSnapshotFromKey(finalizedBundle.ToKey)
	if err != nil {
		return fmt.Errorf("failed getting snapshot height from to_key %s: %w", finalizedBundle.ToKey, err)
	}

	deflated, err := bundles.GetDataFromFinalizedBundle(*finalizedBundle, storageRest)
	if err != nil {
		return fmt.Errorf("failed getting data from finalized bundle: %w", err)
	}

	res, chunks, err := engine.OfferSnapshot(deflated)
	if err != nil {
		return fmt.Errorf("offering snapshot failed: %w", err)
	}

	if res == "ACCEPT" {
		logger.Info().Msg(fmt.Sprintf("offering snapshot for height %d: %s", snapshotHeight, res))
	} else {
		logger.Error().Msg(fmt.Sprintf("offering snapshot for height %d failed: %s", snapshotHeight, res))
		return fmt.Errorf("offering snapshot result: %s", res)
	}

	for chunkIndex := uint32(0); chunkIndex < chunks; chunkIndex++ {
		chunkBundleFinalized, err := bundles.GetFinalizedBundleById(chainRest, snapshotPoolId, snapshotBundleId+int64(chunkIndex))
		if err != nil {
			return fmt.Errorf("failed getting finalized bundle: %w", err)
		}

		chunkBundleDeflated, err := bundles.GetDataFromFinalizedBundle(*chunkBundleFinalized, storageRest)
		if err != nil {
			return fmt.Errorf("failed getting data from finalized bundle: %w", err)
		}

		logger.Info().Msg(fmt.Sprintf("downloaded snapshot chunk %d/%d", chunkIndex+1, chunks))

		res, err := engine.ApplySnapshotChunk(chunkIndex, chunkBundleDeflated)
		if err != nil {
			logger.Error().Msg(fmt.Sprintf("applying snapshot chunk %d/%d failed: %s", chunkIndex+1, chunks, err))
			return err
		}

		if res == "ACCEPT" {
			logger.Info().Msg(fmt.Sprintf("applying snapshot chunk %d/%d: %s", chunkIndex+1, chunks, res))
		} else {
			logger.Error().Msg(fmt.Sprintf("applying snapshot chunk %d/%d failed: %s", chunkIndex+1, chunks, res))
			return fmt.Errorf("applying snapshot chunk: %s", res)
		}
	}

	if err := engine.BootstrapState(deflated); err != nil {
		return fmt.Errorf("failed to bootstrap state: %s\"", err)
	}

	return nil
}
