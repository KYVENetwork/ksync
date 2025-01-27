ARG BASE_IMAGE

FROM $BASE_IMAGE AS build

ARG VERSION
ARG GIT_REPO
ARG DAEMON_NAME
ARG LIBWASM_PATH

WORKDIR /app

RUN apt update && apt upgrade -y

RUN git clone --depth 1 --branch $VERSION $GIT_REPO chain \
    && cd chain \
    && make build \
    && mv build/$DAEMON_NAME /app \
    && cd .. \
    && rm -r chain

RUN if [ -n "$LIBWASM_PATH" ] ; then cp $LIBWASM_PATH /app ; fi

FROM scratch

ARG DAEMON_NAME

COPY --from=build /app /
ENTRYPOINT ["/${DAEMON_NAME}"]
