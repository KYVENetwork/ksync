ARG BASE_IMAGE=golang:1.22
FROM $BASE_IMAGE AS build

WORKDIR /app

RUN apt update && apt upgrade -y

#RUN git clone --depth 1 --branch v1.0.0 https://github.com/CosmWasm/wasmvm.git
#RUN cd wasmvm
#RUN make release-build-macos
#RUN cd ..

#RUN git clone --depth 1 --branch v1.0.0 https://github.com/KYVENetwork/chain.git chain \
#    && cd chain \
#    && env GOOS=${GOOS} GOARCH=${GOARCH} make VERBOSE=1 --debug=j build \
#    && mv build/kyved /bin/chaind \
#    && cd .. \
#    && rm -r chain

RUN git clone --depth 1 --branch v1.0.0 https://github.com/archway-network/archway.git chain \
    && cd chain \
    && make build \
    && mv build/archwayd /bin/chaind \
    && cd .. \
    && rm -r chain

#RUN git clone --depth 1 --branch v10.0.0 https://github.com/osmosis-labs/osmosis.git chain \
#    && cd chain \
#    && env GOOS=${GOOS} GOARCH=${GOARCH} make build \
#    && mv build/osmosisd /bin/chaind \
#    && cd .. \
#    && rm -r chain

FROM scratch
COPY --from=build /bin/chaind /
COPY --from=build "/go/pkg/mod/github.com/!cosm!wasm/wasmvm@v1.2.4/internal/api/libwasmvm.x86_64.so" /
ENTRYPOINT ["/chaind"]
