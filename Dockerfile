FROM kyve/ksync-e2e-tests:latest

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN make build

CMD ["bats", "-T", "--print-output-on-failure", "--verbose-run", "tests"]
