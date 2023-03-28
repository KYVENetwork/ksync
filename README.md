<div align="center">
  <h1>@ksync</h1>
</div>

![banner](assets/ksync.png)

<p align="center">
<strong>Fast Sync archived and validated blocks from KYVE to every Tendermint based Blockchain Application</strong>
</p>

## What is KSYNC?

Since KYVE is validating and archiving blocks from several blockchains permanently this data can be
used to bootstrap nodes. This is especially helpful since most nodes today are pruning nodes and therefore
finding peers which have the requested blocks becomes harder each day. With KSYNC nodes can retrieve
the data from KYVE and directly feed the blocks into every Tendermint based Blockchain Application in order
to sync blocks and join the network.

## How does it work?

KSYNC comes with two sync modes which can be applied depending on the type of application. There is DB-SYNC
which syncs blocks by directly communicating with the app and writing the data directly to the database and then there
P2P-SYNC where KSYNC pretends to be a peer in the network which has all the required blocks, streaming them over
the dedicated block channels over to the node.

After a node has been successfully synced with KSYNC the node simply can fetch remaining blocks and switch to live mode
like it would have if synced normally. This makes operating nodes way cheaper and even may make archival nodes
obsolete since blocks archived by KYVE can then be safely dropped in the nodes and synced again once needed
with this tool.

## Installation

TODO: installation with `go install`

## Usage

Depending on the blockchain application you are trying to sync the following sync modes can be used.

Whichever sync mode you're using you still have to make sure that the blocks for your node are actually available.
You can check out available storage pools for every KYVE network below: 

- **KYVE (Mainnet)**: https://app.kyve.network/#/pools
- **Kaon (Testnet)**: https://app.kaon.kyve.network/#/pools
- **Korellia (Devent)**: https://app.korellia.kyve.network/#/pools

> **_ATTENTION:_**  If you want to use KSYNC for production do not sync blocks from the test- or devnet

### P2P-SYNC

In this sync mode this tool pretends to be a peer which has all the blocks the actual peer node needs. The
blocks are then streamed over the dedicated block channels and storing them is handled by the node itself.

#### Requirements

It does not matter if you want to sync a node from genesis or from an existing height, the following settings have
to be changed in order to run p2p sync.

Make sure that `persistent_peers` are empty in the `config.toml` config file:

`~/.<chain>/config/config.toml`
```toml
[p2p]

persistent_peers = ""
```

Make sure that your `addrbook.json` is empty or delete it entirely:

```bash
rm ~/.<chain>/config/addrbook.json
```

And finally enable the following setting:

`~/.<chain>/config/config.toml`
```toml
[p2p]

allow_duplicate_ip = true
```

#### Sync node

Now you can start your node simply with the `start` command like you would start the node normally. When you see that 
the  node is trying to search for peers but is unable to find any you can start KSYNC.

> **_ATTENTION:_**  If the node actually finds peers the configuration is wrong, in this case double-check the settings
> above

You can then start KSYNC in a **new** terminal with the following command. Please make sure to replace `<user>` and 
`<chain>` with your specific values. This of course is also true for `<pool>` and `<network-api-endpoint>`.

```bash
ksync start mode=p2p --home="/Users/<user>/.<chain>" --pool-id=<pool> --rest=<network-api-endpoint>
```

Available rest endpoints for every network maintained by KYVE:

- **KYVE (Mainnet)**
  - https://eu-api-1.kyve.network
  - https://us-api-1.kyve.network
- **Kaon (Testnet)**
  - https://eu-api-1.kaon.kyve.network
  - https://us-api-1.kaon.kyve.network
- **Korellia (Devent)**
  - https://api.korellia.kyve.network

Once KSYNC starts it automatically continues from the latest height found in the node and starts downloading
the blocks from the storage provider and validates the checksum. You should see blocks streaming over and the node
committing those blocks. If you run this command without a `--target-height` it will sync all blocks which are
available in the pool. You can simply exit the sync process by killing KSYNC with CMD+C.

### DB-SYNC

In this sync mode this tool pretends to be the tendermint process which communicates directly with the
blockchain application over ABCI and replays the blocks against the app and manually writes the results
to the DB directly.

## Live Example: Sync Cosmos Hub over P2P-SYNC

To sync cosmos you have to download and set up the correct gaia binary. To sync from genesis the version `v4.2.1` has
to be used. You can download them [here](https://github.com/cosmos/gaia/releases/tag/v4.2.1) or build them from source: [https://github.com/cosmos/gaia](https://github.com/cosmos/gaia)

Verify installation with

```bash
./gaiad version
4.2.1
```

After the installation init the project

```bash
./gaiad init <your-moniker> --chain-id cosmoshub-4
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

After you see that the node is searching for peers you can start the tool. You can see the latest height which KYVE has
archived [here](https://app.korellia.kyve.network/#/pools/24) under _Latest Key_

```bash
./ksync start --mode=p2p --home="/Users/<user>/.gaia" --pool-id=24 --rest=https://api.korellia.kyve.network
```

You should see the peer connecting and sending over blocks to the gaia node. After all the blocks have been applied
the tool shows _Done_ and you can safely exit the process with CMD+C.

When you want to continue to sync normally you can now add an addrbook or add peers in `persistent_peers`. When you start
the node again the node should continue normally and tries to sync the remaining blocks.
