# KSYNC Architecture Diagram

```mermaid
graph TB
    subgraph "External Data Sources"
        KYVE[KYVE Network API<br/>Bundle Metadata & Pool Info]
        STORAGE[Storage Providers<br/>Arweave, Bundlr, Turbo]
        REGISTRY[Source Registry<br/>GitHub - Chain Configs]
        RPC[RPC Endpoints<br/>Fallback Block Source]
    end

    subgraph "CLI Layer"
        CMD_BLOCK[block-sync]
        CMD_STATE[state-sync]
        CMD_HEIGHT[height-sync]
        CMD_SERVE_SNAP[serve-snapshots]
        CMD_SERVE_BLOCKS[serve-blocks]
        CMD_OTHER[backup/prune/info/reset]
    end

    subgraph "Orchestration Layer"
        BLOCKSYNC[Block Sync Orchestrator<br/>- Bootstrap handling<br/>- Block execution<br/>- Chain upgrades]
        STATESYNC[State Sync Orchestrator<br/>- Snapshot selection<br/>- Chunk application]
        HEIGHTSYNC[Height Sync Orchestrator<br/>- Combined strategy]
        SERVESNAP[Snapshot Server<br/>- HTTP API :7878]
        BOOTSTRAP[Bootstrap Manager<br/>- Genesis handling<br/>- P2P first block]
    end

    subgraph "Data Collection Layer"
        BLOCKS_COL[Blocks Collector<br/>- Stream from bundles<br/>- Checksum validation]
        SNAP_COL[Snapshots Collector<br/>- Find nearest snapshot<br/>- Download chunks]
        BUNDLE_COL[Bundles Collector<br/>- Query KYVE API<br/>- Pagination<br/>- Decompression]
        SOURCE_MGR[Source Manager<br/>- Pool ID mapping<br/>- Version tracking]
    end

    subgraph "Engine Abstraction Layer"
        ENGINE_IF{Engine Interface}
        ENGINE_TM34[Tendermint v0.34<br/>KYVE Fork]
        ENGINE_CB37[CometBFT v0.37]
        ENGINE_CB38[CometBFT v0.38<br/>Default]
        ENGINE_CEL[Celestia Core v0.34]
    end

    subgraph "Binary Process Management"
        BINARY[Application Binary<br/>Cosmos SDK / Custom App]
        ABCI[ABCI Interface<br/>Tendermint Socket Protocol]
        P2P[P2P Interface<br/>First Block Only]
    end

    subgraph "Data Storage"
        BLOCKSTORE[(BlockStore DB<br/>RocksDB)]
        STATESTORE[(State DB<br/>Application State)]
        EVIDENCE[(Evidence Store)]
    end

    subgraph "Supporting Services"
        RPC_SERVER[RPC Server :7777<br/>/status, /block, /block_results]
        BACKUP_MGR[Backup Manager<br/>Compression & Rotation]
        LOGGER[Structured Logger<br/>ZeroLog]
        ANALYTICS[Analytics<br/>Segment - Optional]
    end

    %% External connections
    KYVE -->|Bundle metadata| BUNDLE_COL
    STORAGE -->|Bundle data| BUNDLE_COL
    REGISTRY -->|Chain configs| SOURCE_MGR
    RPC -->|Blocks fallback| BLOCKS_COL

    %% CLI to Orchestration
    CMD_BLOCK --> BLOCKSYNC
    CMD_STATE --> STATESYNC
    CMD_HEIGHT --> HEIGHTSYNC
    CMD_SERVE_SNAP --> SERVESNAP
    CMD_SERVE_BLOCKS --> BLOCKSYNC
    CMD_OTHER --> BACKUP_MGR

    %% Orchestration to Data Collection
    BLOCKSYNC --> BLOCKS_COL
    BLOCKSYNC --> BOOTSTRAP
    STATESYNC --> SNAP_COL
    HEIGHTSYNC --> STATESYNC
    HEIGHTSYNC --> BLOCKSYNC
    SERVESNAP --> BLOCKSYNC

    %% Data Collection internal
    BLOCKS_COL --> BUNDLE_COL
    SNAP_COL --> BUNDLE_COL
    BLOCKS_COL --> SOURCE_MGR
    SNAP_COL --> SOURCE_MGR

    %% Orchestration to Engine
    BLOCKSYNC --> ENGINE_IF
    STATESYNC --> ENGINE_IF
    BOOTSTRAP --> ENGINE_IF

    %% Engine selection
    ENGINE_IF -.->|Auto-select or Manual| ENGINE_TM34
    ENGINE_IF -.->|Auto-select or Manual| ENGINE_CB37
    ENGINE_IF -.->|Auto-select or Manual| ENGINE_CB38
    ENGINE_IF -.->|Auto-select or Manual| ENGINE_CEL

    %% Engine to Binary
    ENGINE_TM34 --> ABCI
    ENGINE_CB37 --> ABCI
    ENGINE_CB38 --> ABCI
    ENGINE_CEL --> ABCI
    ENGINE_TM34 -.->|Genesis >100MB| P2P
    ENGINE_CB37 -.->|Genesis >100MB| P2P
    ENGINE_CB38 -.->|Genesis >100MB| P2P
    ENGINE_CEL -.->|Genesis >100MB| P2P

    %% Binary to Storage
    ABCI <--> BINARY
    P2P <--> BINARY
    BINARY --> BLOCKSTORE
    BINARY --> STATESTORE
    BINARY --> EVIDENCE

    %% Supporting services
    ENGINE_IF --> RPC_SERVER
    ENGINE_IF --> LOGGER
    BLOCKSYNC --> BACKUP_MGR
    BLOCKSYNC --> ANALYTICS

    %% Styling
    classDef external fill:#e1f5ff,stroke:#0288d1,stroke-width:2px
    classDef cli fill:#fff9c4,stroke:#f57f17,stroke-width:2px
    classDef orchestration fill:#f3e5f5,stroke:#7b1fa2,stroke-width:2px
    classDef collection fill:#e8f5e9,stroke:#388e3c,stroke-width:2px
    classDef engine fill:#fce4ec,stroke:#c2185b,stroke-width:2px
    classDef storage fill:#fff3e0,stroke:#e65100,stroke-width:2px
    classDef support fill:#f5f5f5,stroke:#616161,stroke-width:2px

    class KYVE,STORAGE,REGISTRY,RPC external
    class CMD_BLOCK,CMD_STATE,CMD_HEIGHT,CMD_SERVE_SNAP,CMD_SERVE_BLOCKS,CMD_OTHER cli
    class BLOCKSYNC,STATESYNC,HEIGHTSYNC,SERVESNAP,BOOTSTRAP orchestration
    class BLOCKS_COL,SNAP_COL,BUNDLE_COL,SOURCE_MGR collection
    class ENGINE_IF,ENGINE_TM34,ENGINE_CB37,ENGINE_CB38,ENGINE_CEL engine
    class BLOCKSTORE,STATESTORE,EVIDENCE storage
    class RPC_SERVER,BACKUP_MGR,LOGGER,ANALYTICS support
    class BINARY,ABCI,P2P orchestration
```

## Component Descriptions

### External Data Sources
- **KYVE Network API**: Provides bundle metadata, checksums, and pool information
- **Storage Providers**: Host actual block/snapshot data (Arweave, Bundlr, Turbo, KYVE Storage)
- **Source Registry**: GitHub-hosted YAML configs mapping chains to pools and engine versions
- **RPC Endpoints**: Fallback source for blocks when KYVE data unavailable

### CLI Layer
All user-facing commands:
- **block-sync**: Fast sync blocks from KYVE to target height
- **state-sync**: Apply state-sync snapshots from KYVE
- **height-sync**: Combined sync strategy (state-sync → block-sync)
- **serve-snapshots**: Generate and serve snapshots for KYVE pools
- **serve-blocks**: Sync from RPC endpoints instead of KYVE
- **backup/prune/info/reset**: Utility commands

### Orchestration Layer
Business logic coordinators:
- **Block Sync Orchestrator**: Manages block retrieval, execution, and chain upgrades
- **State Sync Orchestrator**: Handles snapshot selection and chunk application
- **Height Sync Orchestrator**: Optimizes sync by combining strategies
- **Snapshot Server**: HTTP API for serving snapshots to other nodes
- **Bootstrap Manager**: Special handling for genesis and first block (P2P when needed)

### Data Collection Layer
- **Blocks Collector**: Streams blocks from bundles with validation
- **Snapshots Collector**: Finds and downloads nearest snapshot chunks
- **Bundles Collector**: Core logic for querying KYVE API and downloading from storage
- **Source Manager**: Manages chain-to-pool mappings and engine version tracking

### Engine Abstraction Layer
Provides unified interface across consensus engine versions:
- **Tendermint v0.34**: KYVE fork for legacy chains
- **CometBFT v0.37**: Mid-version support
- **CometBFT v0.38**: Default modern version
- **Celestia Core v0.34**: Data availability layer support

### Binary Process Management
- **Application Binary**: The actual blockchain application (e.g., Cosmos SDK app)
- **ABCI Interface**: Primary communication via Tendermint Socket Protocol
- **P2P Interface**: Used only for first block when genesis >100MB

### Data Storage
- **BlockStore DB**: Persistent block storage (RocksDB)
- **State DB**: Application state (varies by app)
- **Evidence Store**: Byzantine evidence storage

### Supporting Services
- **RPC Server**: Tendermint-compatible RPC endpoints for monitoring
- **Backup Manager**: Automated backups with compression and rotation
- **Logger**: Structured logging with ZeroLog
- **Analytics**: Optional anonymous usage tracking via Segment
