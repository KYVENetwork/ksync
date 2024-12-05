FROM kyve/ksync-e2e-tests:latest

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN make build
RUN bats -T --print-output-on-failure tests
