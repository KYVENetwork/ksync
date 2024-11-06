FROM golang:1.22-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# test if build succeeds
RUN go build -mod=readonly -o ./build/ksync ./cmd/ksync/main.go

# install kyved
RUN wget -qO- https://github.com/KYVENetwork/chain/releases/download/v1.0.0/kyved_linux_amd64.tar.gz | tar -xzv \
    && mkdir ~/bins \
    && mv kyved ~/bins/kyved-v1.0.0 \
    && ~/bins/kyved-v1.0.0 init test --chain-id kyve-1 \
    && wget https://raw.githubusercontent.com/KYVENetwork/networks/main/kyve-1/genesis.json -O ~/.kyve/config/genesis.json

# run tests
CMD ["go", "test", "-v", "./test/e2e"]
