package helpers

import (
	"fmt"
	"github.com/KYVENetwork/ksync/collectors/bundles"
	"github.com/KYVENetwork/ksync/collectors/pool"
	"github.com/KYVENetwork/ksync/utils"
)

// GetSnapshotPoolHeight returns the height of the snapshot the pool is currently archiving.
// Note that this snapshot can be not complete since for the state-sync to work all chunks have
// to be available.
func GetSnapshotPoolHeight(restEndpoint string, poolId int64) int64 {
	snapshotPool, err := pool.GetPoolInfo(restEndpoint, poolId)
	if err != nil {
		panic(fmt.Errorf("could not get snapshot pool: %w", err))
	}

	var snapshotHeight int64

	if snapshotPool.Pool.Data.CurrentKey == "" {
		snapshotHeight, _, err = utils.ParseSnapshotFromKey(snapshotPool.Pool.Data.StartKey)
		if err != nil {
			panic(fmt.Errorf("could not parse snapshot height from start key: %w", err))
		}
	} else {
		snapshotHeight, _, err = utils.ParseSnapshotFromKey(snapshotPool.Pool.Data.CurrentKey)
		if err != nil {
			panic(fmt.Errorf("could not parse snapshot height from current key: %w", err))
		}
	}

	return snapshotHeight
}

// GetSnapshotBoundaries returns the snapshot heights for the lowest complete snapshot and the
// highest complete snapshot. A complete snapshot contains all chunks of the snapshot, a snapshot which is currently
// still being archived can have the latest chunks missing, therefore being not usable.
// TODO: throw error if no usable snapshot on pool?
func GetSnapshotBoundaries(restEndpoint string, poolId int64) (startSnapshotHeight int64, endSnapshotHeight int64, err error) {
	// load start and latest height
	poolResponse, err := pool.GetPoolInfo(restEndpoint, poolId)
	if err != nil {
		return startSnapshotHeight, endSnapshotHeight, fmt.Errorf("failed to get pool info: %w", err)
	}

	if poolResponse.Pool.Data.Runtime != utils.KSyncRuntimeTendermintSsync {
		return startSnapshotHeight, endSnapshotHeight, fmt.Errorf("found invalid runtime on state-sync pool %d: Expected = %s Found = %s", poolId, utils.KSyncRuntimeTendermintSsync, poolResponse.Pool.Data.Runtime)
	}

	// if no bundles have been created yet or if the current key is empty
	// is no complete snapshot on the pool yet
	if poolResponse.Pool.Data.TotalBundles == 0 || poolResponse.Pool.Data.CurrentKey == "" {
		return startSnapshotHeight, endSnapshotHeight, fmt.Errorf("pool has not produced any bundles yet and therefore has no complete snapshots available")
	}

	startHeight, _, err := utils.ParseSnapshotFromKey(poolResponse.Pool.Data.StartKey)
	if err != nil {
		return startSnapshotHeight, endSnapshotHeight, fmt.Errorf("failed to parse snapshot start key %s: %w", poolResponse.Pool.Data.StartKey, err)
	}

	currentHeight, chunkIndex, err := utils.ParseSnapshotFromKey(poolResponse.Pool.Data.CurrentKey)
	if err != nil {
		return startSnapshotHeight, endSnapshotHeight, fmt.Errorf("failed to parse snapshot current key %s: %w", poolResponse.Pool.Data.CurrentKey, err)
	}

	// if the current height is equal to the start height the pool is still archiving the very first snapshot,
	// therefore the pool has no complete snapshot
	if startHeight == currentHeight {
		return startSnapshotHeight, endSnapshotHeight, fmt.Errorf("pool is still archiving the first snapshot and therefore has no complete snapshots available")
	}

	// if current height is greater than the start height we can be sure that the snapshot at start height
	// is complete
	startSnapshotHeight = startHeight

	// to get the current bundle id we subtract 1 from the total bundles and
	// in order to get the last chunk of the previous complete snapshot we go back
	// all the chunks + 1
	// since it is the goal to get the highest complete snapshot we start at the current bundle id
	// (poolResponse.Pool.Data.TotalBundles - 1) and go back to the snapshot before that since we know
	// that the snapshot before the current one has to be completed (- (chunkIndex +1))
	highestUsableSnapshotBundleId := poolResponse.Pool.Data.TotalBundles - 1 - (chunkIndex + 1)

	bundle, err := bundles.GetFinalizedBundleById(restEndpoint, poolId, highestUsableSnapshotBundleId)
	if err != nil {
		return startSnapshotHeight, endSnapshotHeight, fmt.Errorf("failed to get finalized bundle with id %d: %w", highestUsableSnapshotBundleId, err)
	}

	endSnapshotHeight, _, err = utils.ParseSnapshotFromKey(bundle.ToKey)
	if err != nil {
		return startSnapshotHeight, endSnapshotHeight, fmt.Errorf("failed to parse snapshot key %s: %w", bundle.ToKey, err)
	}

	return
}
