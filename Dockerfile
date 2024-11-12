FROM kyve/ksync-e2e-tests:latest

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# test if build succeeds
RUN go build -mod=readonly -o ./build/ksync ./cmd/ksync/main.go

# run tests
CMD ["bats", "-T", "--print-output-on-failure", "--verbose-run", "tests"]
