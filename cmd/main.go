package main

import (
	"KYVENetwork/kyve-tm-bsync/blocks"
	"KYVENetwork/kyve-tm-bsync/sync"
	"KYVENetwork/kyve-tm-bsync/types"
	"fmt"
)

func main() {
	// needed cli flags
	home := "/Users/troykessler/.kyve"
	pool := int64(0)

	blockCh := make(chan *types.Block)
	quitCh := make(chan int)

	go blocks.NewBundlesReactor(blockCh, quitCh, pool, 0, 0)
	go sync.NewBlockSyncReactor(blockCh, quitCh, home)

	<-quitCh

	fmt.Println("done")
}
