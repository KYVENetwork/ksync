ARG BASE_IMAGE="golang:latest"

FROM $BASE_IMAGE AS build

ARG VERSION
ARG GIT_REPO
ARG BINARY_PATH
ARG LIBWASM_PATH
ARG GO_VERSION
ARG SUBFOLDER
ARG BUILD_CMD="build"

WORKDIR /app

RUN apt update && apt upgrade -y

RUN git clone --depth 1 --branch $VERSION $GIT_REPO repo \
    && cd repo/$SUBFOLDER \
    && env GOOS=darwin GOARCH=arm64 make $BUILD_CMD ENV=mainnet GO_VERSION=$GO_VERSION \
    && mv $BINARY_PATH /app \
    && cd /app \
    && rm -r repo

RUN if [ -n "$LIBWASM_PATH" ] ; then cp $LIBWASM_PATH /app ; fi

FROM scratch

COPY --from=build /app /
