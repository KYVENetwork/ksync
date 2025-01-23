ARG BASE_IMAGE=golang:1.22
FROM $BASE_IMAGE AS build

WORKDIR /app

RUN apt update && apt upgrade -y

RUN git clone --depth 1 --branch v1.0.0 https://github.com/archway-network/archway.git chain \
    && cd chain \
    && make build \
    && mv build/archwayd /app/chaind \
    && cd .. \
    && rm -r chain \
    && cp "/go/pkg/mod/github.com/!cosm!wasm/wasmvm@v1.2.4/internal/api/libwasmvm.x86_64.so" /app

FROM scratch
COPY --from=build /app /
ENTRYPOINT ["/chaind"]
