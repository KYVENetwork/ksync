package db

import (
	"fmt"
	"github.com/KYVENetwork/ksync/collectors/bundles"
	"github.com/KYVENetwork/ksync/collectors/snapshots"
	log "github.com/KYVENetwork/ksync/engines/tendermint"
	"github.com/KYVENetwork/ksync/statesync/helpers"
	"github.com/KYVENetwork/ksync/types"
)

var (
	logger = log.KsyncLogger("state-sync")
)

func StartStateSyncExecutor(engine types.Engine, homePath, chainRest, storageRest string, snapshotPoolId, snapshotHeight int64) error {
	logger.Info().Msg(fmt.Sprintf("applying state-sync snapshot"))

	appHeight, err := engine.GetAppHeight()
	if err != nil {
		return fmt.Errorf("requesting height from app failed: %w", err)
	}

	if appHeight > 0 {
		return fmt.Errorf("app height %d is not zero, please reset with \"ksync unsafe-reset-all\"", appHeight)
	}

	if snapshotHeight == 0 {
		_, _, snapshotHeight, err = helpers.GetSnapshotBoundaries(chainRest, snapshotPoolId)
		if err != nil {
			return fmt.Errorf("failed to get snapshot boundaries: %w", err)
		}

		logger.Info().Msg(fmt.Sprintf("no target height specified, syncing to latest available snapshot %d", snapshotHeight))
	}

	bundleId, err := snapshots.FindBundleIdBySnapshot(chainRest, snapshotPoolId, snapshotHeight)
	if err != nil {
		return fmt.Errorf("error getting bundle id from snapshot: %w", err)
	}

	finalizedBundle, err := bundles.GetFinalizedBundle(chainRest, snapshotPoolId, bundleId)
	if err != nil {
		return fmt.Errorf("failed getting finalized bundle: %w", err)
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
		chunkBundleFinalized, err := bundles.GetFinalizedBundle(chainRest, snapshotPoolId, bundleId+int64(chunkIndex))
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
