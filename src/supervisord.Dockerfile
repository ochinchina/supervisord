FROM golang:alpine AS base
RUN apk add --no-cache --update git gcc rust


FROM base AS builder
COPY . /src
WORKDIR /src
RUN pwd && ls -alh \
 && cd supervisord \
 && CGO_ENABLED=1 go build -a -ldflags "-linkmode external -extldflags -static" -o /usr/local/bin/supervisord \
 && supervisord version


FROM scratch
COPY --from=builder /usr/local/bin/supervisord /usr/local/bin/supervisord
ENTRYPOINT ["/usr/local/bin/supervisord"]
