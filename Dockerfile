FROM golang:1.23-alpine AS app

RUN apk --update --no-cache add gcc musl-dev

WORKDIR /build

COPY src/ .

ARG VERSION

ENV VERSION="${VERSION}"
ENV CGO_ENABLED=1

RUN go build -ldflags="-s -w -X 'github.com/PavelMilanov/container-registry/config.VERSION=${VERSION}'"


FROM node:22 AS web

WORKDIR /app

COPY web/package*.json .

RUN npm ci

COPY web/ .

RUN npm run build


FROM alpine:3.20

ARG USER_DOCKER=registry
ARG UID_DOCKER=10000

ENV USER_DOCKER="$USER_DOCKER"
ENV UID_DOCKER="$UID_DOCKER"

ENV TZ=Europe/Moscow
ENV GIN_MODE=release

WORKDIR /registry

COPY --from=app /build/container-registry /registry/
COPY --from=web /app/dist /registry/

# VOLUME [ "/app/dumps" ]
# VOLUME [ "/app/data" ]

RUN apk --update --no-cache add tzdata sqlite-libs curl && \
    rm -rf /var/cache/apk/ && \
    addgroup -g ${UID_DOCKER} ${USER_DOCKER} && \
    adduser -u ${UID_DOCKER} -G ${USER_DOCKER} -s /bin/sh -D -H ${USER_DOCKER} && \
    chown -R ${USER_DOCKER}:${USER_DOCKER} /registry


EXPOSE 5050/tcp

HEALTHCHECK --interval=1m --timeout=2s --start-period=2s --retries=3 CMD curl -f http://localhost:5050/api/check || exit 1

ENTRYPOINT ["./container-registry" ]

USER ${USER_DOCKER}:${USER_DOCKER}