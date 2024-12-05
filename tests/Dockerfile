FROM golang:1.22

WORKDIR /app

# install latest updates
RUN apt update && apt upgrade -y

# install testing framework "bats" from source
RUN git clone --depth 1 --branch v1.11.0 https://github.com/bats-core/bats-core \
    && cd bats-core \
    && ./install.sh /usr/local \
    && cd ..

# create folder for all binaries
RUN mkdir ~/bins

# install kyved
RUN wget -qO- https://github.com/KYVENetwork/chain/releases/download/v1.0.0/kyved_linux_amd64.tar.gz | tar -xzv \
    && mv kyved ~/bins/kyved-v1.0.0 \
    && ~/bins/kyved-v1.0.0 init ksync --chain-id kyve-1 \
    && wget https://raw.githubusercontent.com/KYVENetwork/networks/main/kyve-1/genesis.json -O ~/.kyve/config/genesis.json

# install cosmovisor with all kyve versions for upgrade testing
RUN go install cosmossdk.io/tools/cosmovisor/cmd/cosmovisor@v1.6.0

ENV DAEMON_HOME=/root/.kyve
ENV DAEMON_NAME=kyved

RUN cosmovisor init ~/bins/kyved-v1.0.0

RUN wget -qO- https://github.com/KYVENetwork/chain/releases/download/v1.1.3/kyved_mainnet_linux_amd64.tar.gz | tar -xzv \
    && cosmovisor add-upgrade v1.1.0 kyved

RUN wget -qO- https://github.com/KYVENetwork/chain/releases/download/v1.2.2/kyved_mainnet_linux_amd64.tar.gz | tar -xzv \
    && cosmovisor add-upgrade v1.2.0 kyved

RUN wget -qO- https://github.com/KYVENetwork/chain/releases/download/v1.3.2/kyved_mainnet_linux_amd64.tar.gz | tar -xzv \
    && cosmovisor add-upgrade v1.3.0 kyved

RUN wget -qO- https://github.com/KYVENetwork/chain/releases/download/v1.4.0/kyved_mainnet_linux_amd64.tar.gz | tar -xzv \
    && cosmovisor add-upgrade v1.4.0 kyved

RUN wget https://github.com/KYVENetwork/chain/releases/download/v1.5.0/kyved_mainnet_linux_amd64 -O kyved \
    && chmod +x kyved \
    && cosmovisor add-upgrade v1.5.0 kyved

# install dydxprotocold
RUN wget -qO- https://github.com/dydxprotocol/v4-chain/releases/download/protocol%2Fv2.0.1/dydxprotocold-v2.0.1-linux-amd64.tar.gz | tar -xzv \
    && mv build/dydxprotocold-v2.0.1-linux-amd64 ~/bins/dydxprotocold-v2.0.1 \
    && ~/bins/dydxprotocold-v2.0.1 init ksync --chain-id dydx-mainnet-1 \
    && wget https://raw.githubusercontent.com/dydxopsdao/networks/main/dydx-mainnet-1/genesis.json -O ~/.dydxprotocol/config/genesis.json

# install archwayd
RUN git clone --depth 1 --branch v1.0.1 https://github.com/archway-network/archway.git \
    && cd archway \
    && make build \
    && mv build/archwayd ~/bins/archwayd-v1.0.1 \
    && ~/bins/archwayd-v1.0.1 init ksync --chain-id archway-1 \
    && wget -qO- https://github.com/archway-network/networks/raw/main/archway/genesis/genesis.json.gz | gunzip > ~/.archway/config/genesis.json \
    && cd .. \
    && rm -r archway

# install celestia-appd
RUN wget -qO- https://github.com/celestiaorg/celestia-app/releases/download/v1.3.0/celestia-app_Linux_x86_64.tar.gz | tar -xzv \
    && mv celestia-appd ~/bins/celestia-appd-v1.3.0 \
    && ~/bins/celestia-appd-v1.3.0 init ksync --chain-id celestia \
    && wget https://raw.githubusercontent.com/celestiaorg/networks/master/celestia/genesis.json -O ~/.celestia-app/config/genesis.json \
    && sed -i -r 's/pyroscope_profile_types = .*/pyroscope_profile_types = ""/' ~/.celestia-app/config/config.toml \
    && rm LICENSE README.md

# install andromedad
RUN git clone --depth 1 --branch v0.1.1-beta-patch https://github.com/andromedaprotocol/andromedad.git \
    && cd andromedad \
    && sed -i -r 's/GO_VERSION := "1.21"/GO_VERSION := "1.22"/' ./Makefile \
    && make build \
    && mv bin/andromedad ~/bins/andromedad-1-v0.1.1-beta-patch \
    && ~/bins/andromedad-1-v0.1.1-beta-patch init ksync --chain-id andromeda-1 \
    && wget https://files.kyve.network/infrastructure/andromeda/genesis.json -O ~/.andromeda/config/genesis.json \
    && cd .. \
    && rm -r andromedad \
    && sed -i -r 's/minimum-gas-prices = ""/minimum-gas-prices = "0uandr"/' ~/.andromeda/config/app.toml

# install nobled
RUN wget https://github.com/noble-assets/noble/releases/download/v8.0.3/nobled_linux-amd64 \
    && chmod +x nobled_linux-amd64 \
    && mv nobled_linux-amd64 ~/bins/nobled-v8.0.3 \
    && ~/bins/nobled-v8.0.3 init ksync --chain-id noble-1 \
    && wget https://raw.githubusercontent.com/strangelove-ventures/noble-networks/main/mainnet/noble-1/genesis.json -O ~/.noble/config/genesis.json
