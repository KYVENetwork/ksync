FROM golang:1.22

WORKDIR /app

RUN apt update
RUN apt upgrade
RUN apt install git
RUN apt install make

# install testing framework "bats" from source
RUN git clone https://github.com/bats-core/bats-core \
    && cd bats-core \
    && ./install.sh /usr/local \
    && cd ..

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# test if build succeeds
RUN go build -mod=readonly -o ./build/ksync ./cmd/ksync/main.go

# create folder for all binaries
RUN mkdir ~/bins

# install kyved
RUN wget -qO- https://github.com/KYVENetwork/chain/releases/download/v1.0.0/kyved_linux_amd64.tar.gz | tar -xzv \
    && mv kyved ~/bins/kyved-v1.0.0 \
    && ~/bins/kyved-v1.0.0 init ksync --chain-id kyve-1 \
    && wget https://raw.githubusercontent.com/KYVENetwork/networks/main/kyve-1/genesis.json -O ~/.kyve/config/genesis.json

# install celestia-appd
RUN wget -qO- https://github.com/celestiaorg/celestia-app/releases/download/v1.3.0/celestia-app_Linux_x86_64.tar.gz | tar -xzv \
    && mv celestia-appd ~/bins/celestia-appd-v1.3.0 \
    && ~/bins/celestia-appd-v1.3.0 init ksync --chain-id celestia \
    && wget https://raw.githubusercontent.com/celestiaorg/networks/master/celestia/genesis.json -O ~/.celestia-app/config/genesis.json \
    && sed -i -r 's/pyroscope_profile_types = .*/pyroscope_profile_types = ""/' ~/.celestia-app/config/config.toml \
    && rm LICENSE README.md

# run tests
CMD ["bats", "--print-output-on-failure", "tests"]
