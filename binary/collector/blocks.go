package collector

import (
	"encoding/json"
	"fmt"
	"github.com/KYVENetwork/ksync/collectors/bundles"
	"github.com/KYVENetwork/ksync/types"
	"github.com/KYVENetwork/ksync/utils"
	tmJson "github.com/tendermint/tendermint/libs/json"
	"strconv"
	"time"
)

type RpcBlockCollector struct {
	rpc                string
	blockRpcReqTimeout time.Duration
	startHeight        int64
	endHeight          int64
}

func NewRpcBlockCollector(rpc string, blockRpcReqTimeout int64) (*RpcBlockCollector, error) {
	result, err := utils.GetFromUrlWithOptions(fmt.Sprintf("%s/status", rpc),
		utils.GetFromUrlOptions{SkipTLSVerification: true},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query rpc endpoint %s: %w", rpc, err)
	}

	var statusResponse types.StatusResponse
	if err := tmJson.Unmarshal(result, &statusResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rpc endpoint response: %w", err)
	}

	return &RpcBlockCollector{
		rpc:                rpc,
		blockRpcReqTimeout: time.Duration(blockRpcReqTimeout * int64(time.Millisecond)),
		startHeight:        statusResponse.Result.SyncInfo.EarliestBlockHeight,
		endHeight:          statusResponse.Result.SyncInfo.LatestBlockHeight,
	}, nil
}

func (collector *RpcBlockCollector) GetStartHeight() int64 {
	return collector.startHeight
}

func (collector *RpcBlockCollector) GetEndHeight() int64 {
	return collector.endHeight
}

func (collector *RpcBlockCollector) GetBlockPair(height int64) ([]byte, []byte, error) {
	firstResponse, err := utils.GetFromUrlWithOptions(fmt.Sprintf("%s/block?height=%d", collector.rpc, height),
		utils.GetFromUrlOptions{SkipTLSVerification: true, WithBackoff: true},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get block %d from rpc: %w", height, err)
	}

	first, err := collector.extractRawBlockFromDataItemValue(firstResponse)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to extract block %d: %w", height, err)
	}

	secondResponse, err := utils.GetFromUrlWithOptions(fmt.Sprintf("%s/block?height=%d", collector.rpc, height+1),
		utils.GetFromUrlOptions{SkipTLSVerification: true, WithBackoff: true},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get block %d from rpc: %w", height+1, err)
	}

	second, err := collector.extractRawBlockFromDataItemValue(secondResponse)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to extract block %d: %w", height+1, err)
	}

	return first, second, nil
}

func (collector *RpcBlockCollector) StreamBlocks(itemCh chan<- types.BlockItem, errorCh chan<- error, continuationHeight, targetHeight int64, exitOnTargetHeight bool) {
	for {
		// TODO: log here?
		// logger.Info().Msg(fmt.Sprintf("downloading block with height %d", continuationHeight))

		blockResponse, err := utils.GetFromUrlWithOptions(fmt.Sprintf("%s/block?height=%d", collector.rpc, continuationHeight),
			utils.GetFromUrlOptions{SkipTLSVerification: true, WithBackoff: true},
		)
		if err != nil {
			errorCh <- fmt.Errorf("failed to get block %d from rpc: %w", continuationHeight, err)
			return
		}

		block, err := collector.extractRawBlockFromDataItemValue(blockResponse)
		if err != nil {
			errorCh <- fmt.Errorf("failed to extract block %d: %w", continuationHeight, err)
			return
		}

		itemCh <- types.BlockItem{
			Height: continuationHeight,
			Block:  block,
		}

		if exitOnTargetHeight && targetHeight > 0 && continuationHeight >= targetHeight+1 {
			break
		}

		continuationHeight++
		time.Sleep(collector.blockRpcReqTimeout)
	}
}

func (collector *RpcBlockCollector) extractRawBlockFromDataItemValue(value []byte) ([]byte, error) {
	var block struct {
		Result struct {
			Block json.RawMessage `json:"value"`
		} `json:"result"`
	}

	if err := json.Unmarshal(value, &block); err != nil {
		return nil, fmt.Errorf("failed to unmarshal block response: %w", err)
	}

	return block.Result.Block, nil
}

type KyveBlockCollector struct {
	poolId      int64
	runtime     string
	chainRest   string
	storageRest string
	startHeight int64
	endHeight   int64
}

func NewKyveBlockCollector(poolId int64, chainRest, storageRest string) (*KyveBlockCollector, error) {
	data, err := utils.GetFromUrlWithBackoff(fmt.Sprintf("%s/kyve/query/v1beta1/pool/%d", chainRest, poolId))
	if err != nil {
		return nil, fmt.Errorf("failed to query pool %d", poolId)
	}

	var poolResponse types.PoolResponse

	if err = json.Unmarshal(data, &poolResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pool response: %w", err)
	}

	startHeight, err := strconv.ParseInt(poolResponse.Pool.Data.StartKey, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse start height %s from pool: %w", poolResponse.Pool.Data.StartKey, err)
	}

	endHeight, err := strconv.ParseInt(poolResponse.Pool.Data.CurrentKey, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse end height %s from pool: %w", poolResponse.Pool.Data.CurrentKey, err)
	}

	return &KyveBlockCollector{
		poolId:      poolId,
		runtime:     poolResponse.Pool.Data.Runtime,
		chainRest:   chainRest,
		storageRest: storageRest,
		startHeight: startHeight,
		endHeight:   endHeight,
	}, nil
}

func (collector *KyveBlockCollector) GetStartHeight() int64 {
	return collector.startHeight
}

func (collector *KyveBlockCollector) GetEndHeight() int64 {
	return collector.endHeight
}

func (collector *KyveBlockCollector) GetBlockPair(height int64) ([]byte, []byte, error) {
	return nil, nil, nil
}

func (collector *KyveBlockCollector) StreamBlocks(itemCh chan<- types.BlockItem, errorCh chan<- error, continuationHeight, targetHeight int64, exitOnTargetHeight bool) {
	paginationKey, err := getPaginationKeyForBlockHeight(collector.chainRest, blockPool, continuationHeight)
	if err != nil {
		errorCh <- fmt.Errorf("failed to get pagination key for continuation height %d: %w", continuationHeight, err)
		return
	}

BundleCollector:
	for {
		bundlesPage, nextKey, err := bundles.GetFinalizedBundlesPage(collector.chainRest, collector.poolId, utils.BundlesPageLimit, paginationKey, false)
		if err != nil {
			errorCh <- fmt.Errorf("failed to get finalized bundles page: %w", err)
			return
		}

		for _, finalizedBundle := range bundlesPage {
			// TODO: rename this height to maxPageHeight or something
			height, err := strconv.ParseInt(finalizedBundle.ToKey, 10, 64)
			if err != nil {
				errorCh <- fmt.Errorf("failed to parse bundle to key to int64: %w", err)
				return
			}

			if height < continuationHeight {
				continue
			}

			// TODO: log here?
			// logger.Info().Msg(fmt.Sprintf("downloading bundle with storage id %s", finalizedBundle.StorageId))

			deflated, err := bundles.GetDataFromFinalizedBundle(finalizedBundle, collector.storageRest)
			if err != nil {
				errorCh <- fmt.Errorf("failed to get data from finalized bundle: %w", err)
				return
			}

			// parse bundle
			var bundle types.Bundle

			if err := json.Unmarshal(deflated, &bundle); err != nil {
				errorCh <- fmt.Errorf("failed to unmarshal tendermint bundle: %w", err)
				return
			}

			for _, dataItem := range bundle {
				itemHeight, err := utils.ParseBlockHeightFromKey(dataItem.Key)
				if err != nil {
					errorCh <- fmt.Errorf("failed parse block height from key %s: %w", dataItem.Key, err)
					return
				}

				// skip blocks until we reach start height
				if itemHeight < continuationHeight {
					continue
				}

				block, err := collector.extractRawBlockFromDataItemValue(dataItem.Value)
				if err != nil {
					errorCh <- fmt.Errorf("failed to extract block %d from data item value: %w", itemHeight, err)
				}

				// send raw data item executor
				itemCh <- types.BlockItem{
					Height: continuationHeight,
					Block:  block,
				}

				// keep track of latest retrieved height
				continuationHeight = itemHeight + 1

				// exit if mustExit is true and target height is reached
				if exitOnTargetHeight && targetHeight > 0 && itemHeight >= targetHeight+1 {
					break BundleCollector
				}
			}
		}

		if nextKey == "" {
			// if there is no new page we do not continue
			if exitOnTargetHeight {
				break
			}

			// if we are at the end of the page we continue and wait for
			// new finalized bundles
			time.Sleep(30 * time.Second)
			continue
		}

		time.Sleep(utils.RequestTimeoutMS)
		paginationKey = nextKey
	}
}

func (collector *KyveBlockCollector) extractRawBlockFromDataItemValue(value []byte) ([]byte, error) {
	if collector.runtime == utils.KSyncRuntimeTendermint {
		var block struct {
			Block struct {
				Block json.RawMessage `json:"block"`
			} `json:"block"`
		}

		if err := json.Unmarshal(value, &block); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tendermint data item: %w", err)
		}

		return block.Block.Block, nil
	}

	if collector.runtime == utils.KSyncRuntimeTendermintBsync {
		return value, nil
	}

	return nil, fmt.Errorf("unknown runtime %s", collector.runtime)
}

// getPaginationKeyForBlockHeight gets the pagination key right for the bundle so the StartBlockCollector can
// directly start at the correct bundle. Therefore, it does not need to search through all the bundles until
// it finds the correct one
func getPaginationKeyForBlockHeight(chainRest string, blockPool types.PoolResponse, height int64) (string, error) {
	finalizedBundle, err := bundles.GetFinalizedBundleForBlockHeight(chainRest, blockPool, height)
	if err != nil {
		return "", fmt.Errorf("failed to get finalized bundle for block height %d: %w", height, err)
	}

	bundleId, err := strconv.ParseInt(finalizedBundle.Id, 10, 64)
	if err != nil {
		return "", fmt.Errorf("failed to parse bundle id %s: %w", finalizedBundle.Id, err)
	}

	// if bundleId is zero we start from the beginning, meaning the paginationKey should be empty
	if bundleId == 0 {
		return "", nil
	}

	_, paginationKey, err := bundles.GetFinalizedBundlesPageWithOffset(chainRest, blockPool.Pool.Id, 1, bundleId-1, "", false)
	if err != nil {
		return "", fmt.Errorf("failed to get finalized bundles: %w", err)
	}

	return paginationKey, nil
}
