package main

import (
	"KYVENetwork/kyve-tm-bsync/blocks"
	"KYVENetwork/kyve-tm-bsync/sync"
	"KYVENetwork/kyve-tm-bsync/types"
	"fmt"
)

func main() {
	homeDir := "/Users/troykessler/.kyve"

	blockCh := make(chan *types.Block)
	quitCh := make(chan int)

	go blocks.NewBundlesReactor(blockCh, quitCh, 0, 0, 0)
	go sync.NewBlockSyncReactor(blockCh, quitCh, homeDir)

	<-quitCh

	fmt.Println("done")
}
