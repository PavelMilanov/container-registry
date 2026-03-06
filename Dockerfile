# Stage 1
FROM golang:1.26-alpine AS app

RUN apk --update --no-cache add gcc musl-dev

WORKDIR /build

COPY src/go.mod .

RUN go mod download

COPY src/ .

ARG VERSION

ENV VERSION="${VERSION}"
ENV CGO_ENABLED=1

RUN go install -trimpath -ldflags="-s -w -X 'github.com/PavelMilanov/container-registry/config.VERSION=${VERSION}'"


# Stage 2
FROM alpine:3.23

ENV TZ=Europe/Moscow
ENV GIN_MODE=release
ENV USER=registry
ENV UID=10000

RUN apk --update --no-cache add tzdata sqlite-libs

RUN addgroup -g ${UID} ${USER} && \
    adduser -u ${UID} -G ${USER} -s /bin/sh -D -H ${USER}


WORKDIR /registry

COPY --from=app /go/bin/container-registry /usr/bin/cr

RUN chmod +x /usr/bin/cr

EXPOSE 5050/tcp

USER ${USER}

HEALTHCHECK --interval=10m --timeout=5s --start-period=5s --retries=3 CMD ["/usr/bin/cr", "healthcheck"]

ENTRYPOINT [ "/usr/bin/cr" ]

CMD ["serve" ]
