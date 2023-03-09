package sync

import (
	"KYVENetwork/kyve-tm-bsync/types"
	"fmt"
)

func NewBlockSyncReactor(blockCh <-chan *types.Block, quitCh <-chan int) {
	for {
		select {
		case block := <-blockCh:
			fmt.Printf("%v %s\n", block.Header.Height, block.Header.AppHash)
		case <-quitCh:
			return
		}
	}
}
