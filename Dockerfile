FROM golang:1.24-alpine AS app

RUN apk --update --no-cache add gcc musl-dev

WORKDIR /

COPY src/go.mod src/go.sum ./

RUN go mod download && go mod verify

COPY src/ ./

ARG VERSION

ENV VERSION="${VERSION}"
ENV CGO_ENABLED=1

RUN go install -ldflags="-s -w -X 'config.VERSION=${VERSION}'"


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

COPY --from=app /go/bin/container-registry /registry/registry
COPY --from=web /app/dist /registry/

RUN apk --update --no-cache add tzdata sqlite-libs curl && \
    addgroup -g ${UID_DOCKER} ${USER_DOCKER} && \
    adduser -u ${UID_DOCKER} -G ${USER_DOCKER} -s /bin/sh -D -H ${USER_DOCKER} && \
    chown -R ${USER_DOCKER}:${USER_DOCKER} /registry

EXPOSE 5050/tcp

HEALTHCHECK --interval=10m --timeout=3s --start-period=5s --retries=3 CMD curl -f http://localhost:5050/check || exit 1

ENTRYPOINT ["./registry" ]

USER ${USER_DOCKER}:${USER_DOCKER}
