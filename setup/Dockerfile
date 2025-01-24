ARG BASE_IMAGE

FROM $BASE_IMAGE AS build

ARG VERSION
ARG GIT_REPO
ARG DAEMON_NAME

WORKDIR /app

RUN apt update && apt upgrade -y

RUN git clone --depth 1 --branch $VERSION $GIT_REPO chain \
    && cd chain \
    && make build \
    && mv build/$DAEMON_NAME /app \
    && cd .. \
    && rm -r chain \
    && cp "/go/pkg/mod/github.com/!cosm!wasm/wasmvm@v1.2.4/internal/api/libwasmvm.x86_64.so" /app

#RUN git clone --depth 1 --branch v10.0.0 https://github.com/osmosis-labs/osmosis chain \
#    && cd chain \
#    && make build \
#    && mv build/osmosisd /app/chaind \
#    && cd .. \
#    && rm -r chain \
#    && cp "/go/pkg/mod/github.com/!cosm!wasm/wasmvm@v1.0.0/api/libwasmvm.x86_64.so" /app

FROM scratch

ARG DAEMON_NAME

COPY --from=build /app /
ENTRYPOINT ["/${DAEMON_NAME}"]
