package helpers

import (
	"fmt"
	"github.com/KYVENetwork/ksync/collectors/bundles"
	"github.com/KYVENetwork/ksync/collectors/pool"
	"github.com/KYVENetwork/ksync/utils"
	"strconv"
	"strings"
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
func GetSnapshotBoundaries(restEndpoint string, poolId int64) (startHeight int64, endHeight int64, err error) {
	// load start and latest height
	poolResponse, err := pool.GetPoolInfo(restEndpoint, poolId)
	if err != nil {
		return startHeight, endHeight, fmt.Errorf("failed to get pool info: %w", err)
	}

	if poolResponse.Pool.Data.Runtime != utils.KSyncRuntimeTendermintSsync {
		return startHeight, endHeight, fmt.Errorf("found invalid runtime on state-sync pool %d: Expected = %s Found = %s", poolId, utils.KSyncRuntimeTendermintSsync, poolResponse.Pool.Data.Runtime)
	}

	// if no bundles have been created yet or if the current key is empty
	// is no complete snapshot on the pool yet
	if poolResponse.Pool.Data.TotalBundles == 0 || poolResponse.Pool.Data.CurrentKey == "" {
		return startHeight, endHeight, fmt.Errorf("pool has not produced any bundles yet and therefore has no complete snapshots available")
	}

	startHeight, _, err = utils.ParseSnapshotFromKey(poolResponse.Pool.Data.StartKey)
	if err != nil {
		return startHeight, endHeight, fmt.Errorf("failed to parse snapshot start key %s: %w", poolResponse.Pool.Data.StartKey, err)
	}

	_, chunkIndex, err := utils.ParseSnapshotFromKey(poolResponse.Pool.Data.CurrentKey)
	if err != nil {
		return startHeight, endHeight, fmt.Errorf("failed to parse snapshot current key %s: %w", poolResponse.Pool.Data.CurrentKey, err)
	}

	// to get the current bundle id we subtract 1 from the total bundles and
	// in order to get the last chunk of the previous complete snapshot we go back
	// all the chunks + 1
	// since it is the goal to get the highest complete snapshot we start at the current bundle id
	// (poolResponse.Pool.Data.TotalBundles - 1) and go back to the snapshot before that since we know
	// that the snapshot before the current one has to be completed (- (chunkIndex +1))
	highestUsableSnapshotBundleId := poolResponse.Pool.Data.TotalBundles - 1 - (chunkIndex + 1)

	// check if the current chunk is the last chunk of the snapshot, if yes we highestUsableSnapshotBundleId
	// is the current one. we check this by comparing the number of chunks from the bundle summary with the
	// current chunk index from the current key.
	summary := strings.Split(poolResponse.Pool.Data.CurrentSummary, "/")
	// bundle summary format is "height/format/chunkIndex/chunks"
	// if the summary does not have the right format we skip because this is probably a legacy bundle summary.
	if len(summary) == 4 {
		if chunks, _ := strconv.ParseInt(summary[3], 10, 64); chunks == chunkIndex+1 {
			highestUsableSnapshotBundleId = poolResponse.Pool.Data.TotalBundles - 1
		}
	}

	bundle, err := bundles.GetFinalizedBundleById(restEndpoint, poolId, highestUsableSnapshotBundleId)
	if err != nil {
		return startHeight, endHeight, fmt.Errorf("failed to get finalized bundle with id %d: %w", highestUsableSnapshotBundleId, err)
	}

	endHeight, _, err = utils.ParseSnapshotFromKey(bundle.ToKey)
	if err != nil {
		return startHeight, endHeight, fmt.Errorf("failed to parse snapshot key %s: %w", bundle.ToKey, err)
	}

	return
}
