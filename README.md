<div align="center">
  <h1>@ksync</h1>
</div>

![banner](assets/ksync.png)

<p align="center">
<strong>Rapidly sync validated blocks and snapshots from KYVE to every Tendermint based Blockchain Application</strong>
</p>

## Content

- [What is KSYNC?](#what-is-ksync)
- [How does it work?](#how-does-it-work)
- [Installation](#installation)
- [Usage](#usage)
  - [Limitations](#limitations)
  - [BLOCK-SYNC](#block-sync)
  - [STATE-SYNC](#state-sync)
  - [HEIGHT-SYNC](#height-sync)
- [For node runners](#for-node-runners)
  - [SERVE-SNAPSHOTS](#serve-snapshots)
- [Examples](#examples)

## What is KSYNC?

Since KYVE is validating and archiving blocks and state-sync snapshots from several blockchains permanently this data can be
used to bootstrap nodes. This is especially helpful since most nodes today are pruning nodes and therefore
finding peers which have the requested blocks becomes harder each day. With KSYNC nodes can retrieve
the data from KYVE and directly feed the blocks into every Tendermint based Blockchain Application in order
to sync blocks and join the network. Furthermore, any Tendermint based application can rapidly join the network by 
applying state-sync snapshots which are permanently archived on Arweave.

## How does it work?

KSYNC basically replaces the inbuilt tendermint process and communicates with the app directly over the Tendermint
Socket Protocol (TSP) with the [ABCI](https://github.com/tendermint/spec/blob/master/spec/abci/abci.md) interface.
Once KSYNC has retrieved the requested blocks for the application from a permanent storage provider like Arweave it
executes them against the app and stores all relevant information in the blockstore and state.db databases directly. The
same applies to state-sync snapshots, where KSYNC offers the snapshots over the ABCI methods against the app.

After a node has been successfully synced with KSYNC the node simply can fetch remaining blocks and switch to live mode
like it would have if synced normally. This makes operating nodes way cheaper and even may make archival nodes
obsolete since blocks archived by KYVE can then be safely dropped in the nodes and synced again once needed
with this tool.

Overview of how KSYNC interacts with the tendermint application:

<p align="center">
  <img width="70%" src="assets/db_sync.png" />
</p>

## Installation

To install the latest version of `ksync`, run the following command:

```bash
go install github.com/KYVENetwork/ksync/cmd/ksync@latest
```

To install a previous version, you can specify the version.

```bash
go install github.com/KYVENetwork/ksync/cmd/ksync@v0.5.0
```

Run `ksync version` to verify the installation.

You can also install from source by pulling the ksync repository and switching to the correct version and building
as follows:

```bash
git clone git@github.com:KYVENetwork/ksync.git
cd ksync
git checkout tags/vx.x.x -b vx.x.x
make ksync
```

This will build ksync in `/build` directory. Afterwards, you may want to put it into your machine's PATH like
as follows:

```bash
cp build/ksync ~/go/bin/ksync
```

## Usage

Depending on what you want to achieve with KSYNC there are three sync modes available. A quick summary of what they do
and when to use them can be found below:

- **block-sync**
  - Syncs blocks from the nodes current height up to a specified target height. With this the node has stored and checked every block.
  - Generally recommended for archival node runners, who want to have a full node containing all blocks.
- **state-sync**
  - Applies a state-sync snapshot to the node. After the snapshot got applied the node can continue block-syncing from the applied snapshot height.
  - Generally recommended for new node runners, who want to join a network in minutes without wanting to sync the entire blockchain.
- **height-sync**
  - Finds the quickest way out of state-sync and height-sync to get to the specified target height.
  - Generally recommended for users who want to check out a historical state within minutes at the specified target height for e.g. analysis.

### Limitations

Because KSYNC uses the blocks and snapshots archived by the KYVE storage pools you first have to check if those pools
are available in the first place for your desired chain and block height.

Depending on the KYVE network, you can find all available data pools here:

- **Mainnet (KYVE)**: https://app.kyve.network/#/pools
- **Testnet (Kaon)**: https://app.kaon.kyve.network/#/pools
- **Devnet (Korellia)**: https://app.korellia.kyve.network/#/pools

Depending on the sync mode you use, the data pools need to run on the following runtimes:

- **block-sync**: `@kyvejs/tendermint` or `@kyvejs/tendermint-bsync`
- **state-sync**: `@kyvejs/tendermint-ssync`
- **height-sync**: `@kyvejs/tendermint` or `@kyvejs/tendermint-bsync` and `@kyvejs/tendermint-ssync`

### BLOCK-SYNC

#### Syncing to latest available height

Depending on your current node height (can be also 0  if you start syncing from genesis) you can sync up to the latest
height available by the storage pool. KSYNC will automatically exit once that height is reached.

```bash
ksync height-sync --binary="/path/to/<binaryd>" --home="/path/to/.<home>" --block-pool-id=<pool-id>
```

#### Syncing to specified target height

Depending on your current node height (can be also 0  if you start syncing from genesis) you can sync up to your desired
target height. KSYNC will automatically exit once that height is reached.

```bash
ksync height-sync --binary="/path/to/<binaryd>" --home="/path/to/.<home>" --block-pool-id=<pool-id> --target-height=<height>
```

### STATE-SYNC

#### Syncing to latest available snapshot height

You can state-sync a node if it has no height (either node has to be just initialized or reset with `ksync unsafe-reset-all`)
to the latest available snapshot archived by the pool with the following command. If the storage pool has synced with the live
height this can be used to rapidly join this network.

```bash
ksync state-sync --binary="/path/to/<binaryd>" --home="/path/to/.<home>" --snapshot-pool-id=<pool-id>
```

#### Syncing to specified snapshot height

You can state-sync a node if it has no height (either node has to be just initialized or reset with `ksync unsafe-reset-all`)
to your desired target height. The target height has to be the exact height of the archived snapshot. If the specified
height can not be found it prints out the nearest available snapshot height you can use.

```bash
ksync state-sync --binary="/path/to/<binaryd>" --home="/path/to/.<home>" --snapshot-pool-id=<pool-id> --target-height=<height>
```

### HEIGHT-SYNC

#### Syncing to latest available block height

You can height-sync a node if it has no height (either node has to be just initialized or reset with `ksync unsafe-reset-all`)
to the latest available height. This is especially useful for joining a new network if the user wants to join as quick as
possible.

```bash
ksync height-sync --binary="/path/to/<binaryd>" --home="/path/to/.<home>" --snapshot-pool-id=<pool-id> --block-pool-id=<pool-id>
```

#### Syncing to specified target height

You can height-sync a node if it has no height (either node has to be just initialized or reset with `ksync unsafe-reset-all`)
to your desired target height. The target height can be any height (but the block data pool must have archived it), then
it will use available state-sync snapshots and block-sync to get to the target height as quickly as possible

```bash
ksync height-sync --binary="/path/to/<binaryd>" --home="/path/to/.<home>" --snapshot-pool-id=<pool-id> --block-pool-id=<pool-id> --target-height=<height>
```

## For node runners

This section includes all commands used by KYVE protocol node operators to participate in state-sync data pools.

### SERVE-SNAPSHOTS

This command is essential for running as a protocol node in a state-sync pool since this will serve the snapshots to the
protocol node. Basically, KSYNC will sync the blocks with block-sync and waits for the ABCI app to create the snapshots,
once created they are exposed over a REST API server which the protocol node can then query.

To start with default settings serve the snapshots with:

```bash
ksync serve-snapshots --binary="/path/to/<binaryd>" --home="/path/to/.<home>" --snapshot-pool-id=<pool-id> --block-pool-id=<pool-id>
```

Once you see that KSYNC is syncing blocks you can open `https://localhost:7878/list_snapshots`. In the beginning it should
return an empty array, but after the first snapshot height is reached (check the interval in the data pool settings) you 
should see a first snapshot object in the response.

#### Changing snapshot api server port

You can change the snapshot api server port with the flag `--snapshot-port=<port>`

#### Enabling metrics server and manage port

You can enable a metrics server running by default on `http://localhost:8080/metrics` by add the flag `--metrics`.
Furthermore, can you change the port of the metrics server by adding the flag `--metrics-port=<port>`

#### Manage pruning

By default, pruning is enabled. That means that all blocks, states and snapshots prior to the snapshot pool height
are automatically, deleted, saving a lot of disk space. If you want to disable it add the flag `--pruning=false`

## Examples

Coming soon

## Backups

Even with the right setup and careful maintenance, it's possible to encounter app-hash errors or other unexpected problems that can lead to node collisions. Especially when you're dealing with syncing an archival node, it's a good idea to create periodic backups of the node's data.

KSYNC offers precisely this option for creating backups. There are two different methods to utilize this:

### 1. BLOCK-SYNC-Backups

With BLOCK-SYNC, nodes can be synced by KSYNC from any height up to the latest height available by the storage pool.
Backups can be created automatically at an interval, with the following parameters:

```bash
--home                 string   'home directory of the node (e.g. ~/.osmosisd)'
--backup-interval      int      'block interval to write backups of data directory (set 0 to disable backups)'
--backup-keep-recent   int      'number of latest backups to be keep (0 to keep all backups)'
--backup-compression   string   'compression type used for backups ("tar.gz","zip"), if not compression given the backup will be stored uncompressed'
--backup-dest          string   'path where backups should be stored [default = ~/.ksync/backups]'
```

When the specified `backup-interval` is reached (`height % backup-interval = 0`), KSYNC temporarily pauses the sync process and creates a backup. 
These backups are duplicates of the node's data directory (e.g. `~/.osmosisd/data`). If compression is enabled (e.g. using `--backup-compression="tar.gz"`), the backup is compressed and the original uncompressed version is deleted after successful compression in a parallel process.

#### Usage 

Because backups are disabled by default, it's only required to set ``backup-interval``, whereas the other flags are optional.
Since the creation of a backup takes steadily longer as the data size grows, it is recommended to choose an interval of more than `20000` blocks.

Example command to run BLOCK-SYNC with compressed backups:
```bash
ksync block-sync --binary="/path/to/<binaryd>" --home="/path/to/.<home>" --block-pool-id=<pool-id> --target-height=<height>
  --backup-interval=50000 --backup-compression="tar.gz"
```

### 2. Backup-Command

The backup functionality can of course also be used with a standalone command. In this case everything runs in one process
where the following flags can be used:

```bash
--home                 string   'home directory of the node (e.g. ~/.osmosisd)'
--backup-keep-recent   int      'number of latest backups to be keep (0 to keep all backups)'
--backup-compression   string   'compression type used for backups ("tar.gz","zip"), if not compression given the backup will be stored uncompressed'
--backup-dest          string   'path where backups should be stored [default = ~/.ksync/backups]'
```

#### Usage

```bash
ksync backup --home="/Users/christopher/.osmosisd" --compression="tar.gz"
```