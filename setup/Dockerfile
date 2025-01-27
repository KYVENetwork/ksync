ARG BASE_IMAGE="golang:latest"

FROM $BASE_IMAGE AS build

ARG VERSION
ARG GIT_REPO
ARG DAEMON_NAME
ARG LIBWASM_PATH
ARG GO_VERSION

WORKDIR /app

RUN apt update && apt upgrade -y

RUN git clone --depth 1 --branch $VERSION $GIT_REPO chain \
    && cd chain \
    && make build ENV=mainnet GO_VERSION=$GO_VERSION \
    && mv build/$DAEMON_NAME /app \
    && cd .. \
    && rm -r chain

RUN if [ -n "$LIBWASM_PATH" ] ; then cp $LIBWASM_PATH /app ; fi

FROM scratch

ARG DAEMON_NAME

COPY --from=build /app /
ENTRYPOINT ["/${DAEMON_NAME}"]
