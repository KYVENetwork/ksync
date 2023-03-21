# KSYNC

Fast Sync validated and archived blocks from KYVE to every Tendermint based Blockchain Application

## What is KSYNC?

KYVE plans to validate and archive block data of several blockchains including those of the cosmos ecosystem. For a first
real world usecase of this data this script is intended for syncing blocks from KYVE directly into the node so it can take
part in consenus once fully synced. This may make archival nodes obsolete since all historical blocks can then be archived
with KYVE and can later be retrieved with this script.

## How does it work?

This script acts and mimics a peer in the network. When the node starts the addrbook and any peers in the config should be empty, therefore looking
for new peers it can connect to. When this script starts it mimics a peer which has all the blocks and tries to connect with the node. Once the node accepts the new peer it requests
those blocks, which the script pulls down from Arweave, checks the checksum of KYVE and forwards them through the p2p channels. After the node has synced blocks with 
KYVE it can connect to other, "real" peers and continue on consensus or pull down remaining blocks.

## Important

**ATTENTION: This is still WIP and only intended as a POC**

This script works with Cosmos Hub, Evmos, Axelar and Osmosis. Two examples of how to sync node with KYVE
data are shown below. This is **not** production ready and future plans are making are further developing this script
so it can be used by every cosmos chain to sync historical or recent blocks. Further plans are to include state sync
so nodes can join networks even faster without worrying if the snapshots are valid or available.

**NOTE**: After a sync the node can continue normally and take part in consensus or futher fast sync remaining blocks.
For now to restart the sync the data folder has to be cleared. For that the command `./bin unsafe-reset-all` can be used.

## Install

Checkout repository and build binary

```bash
git clone https://github.com/KYVENetwork/ksync.git
cd ksync
go build -o ksync cmd/ksync/main.go
```

For every example the ulimit has to be increased with `ulimit -n 16384`

## Example 1: Sync Osmosis

To sync osmosis you have to download and set up the correct osmosis binary. To sync from genesis the version `v3.1.0` has
to be used. You can download them [here](https://github.com/osmosis-labs/osmosis/releases/tag/v3.1.0) or build them from source: [https://github.com/osmosis-labs/osmosis](https://github.com/osmosis-labs/osmosis)

Verify installation with

```bash
./osmosisd version
3.1.0
```

After the installation init the config

```bash
./osmosisd init <your-moniker> --chain-id osmosis-1
```

download the genesis

```bash
wget -O ~/.osmosisd/config/genesis.json https://github.com/osmosis-labs/networks/raw/main/osmosis-1/genesis.json
```

and edit the following in `~/.osmosisd/config/config.toml`. TIP: those settings can be found under "p2p"

```toml
allow_duplicate_ip = true
```

Important: Don't include an addrbook.json and make sure persistent_peers and etc. are empty for now or else the node will connect to other peers. It should only connect
to our peer.

when the config is done the node can be started

```bash
./osmosisd start
```

After you see that the node is searching for peers you can open a new terminal and start ksync. For testing purposes we validated and
archived the first 600 blocks of the chain `osmosis-1`

```bash
./ksync start --mode=p2p --home="/Users/<user>/.osmosisd" --pool-id=3 --rest=http://35.158.99.65:1317
```

**INFO**: The rest endpoint points to a local KYVE chain running with some testdata

You should see the peer connecting and sending over blocks to the osmosis node. After the 600 blocks were applied
you can abort both processes with CMD+C

When you want to continue to sync normally you can now add an addrbook or add peers in `persistent_peers`. When you start
the node again the node should continue normally and tries to sync the remaining blocks.

## Example 2: Sync Cosmos

To sync cosmos you have to download and set up the correct gaia binary. To sync from genesis the version `v4.2.1` has
to be used. You can download them [here](https://github.com/cosmos/gaia/releases/tag/v4.2.1) or build them from source: [https://github.com/cosmos/gaia](https://github.com/cosmos/gaia)

Verify installation with

```bash
./gaiad version
4.2.1
```

After the installation init the config

```bash
./gaiad init <your-moniker>
```

download the genesis

```bash
wget https://raw.githubusercontent.com/cosmos/mainnet/master/genesis/genesis.cosmoshub-4.json.gz
gzip -d genesis.cosmoshub-4.json.gz
mv genesis.cosmoshub-4.json ~/.gaia/config/genesis.json
```

and edit the following in `~/.gaia/config/config.toml`. TIP: those settings can be found under "p2p"

```toml
allow_duplicate_ip = true
```

Important: Don't include an addrbook.json and make sure persistent_peers and etc. are empty for now or else the node will connect to other peers. It should only connect
to our peer.

when the config is done the node can be started. NOTE: this can take a while (~5mins) since the genesis file is quite big.

```bash
./gaiad start --x-crisis-skip-assert-invariants
```

After you see that the node is searching for peers you can start the script. For testing purposes we validated and
archived the first 5000 blocks of the chain `cosmoshub-4` 

```bash
./ksync start --mode=p2p --home="/Users/<user>/.gaia" --pool-id=0 --rest=http://35.158.99.65:1317
```

**INFO**: The rest endpoint points to a local KYVE chain running with some testdata

You should see the peer connecting and sending over blocks to the gaia node. After the 5000 blocks were applied
you can abort both processes with CMD+C

When you want to continue to sync normally you can now add an addrbook or add peers in `persistent_peers`. When you start
the node again the node should continue normally and tries to sync the remaining blocks.
