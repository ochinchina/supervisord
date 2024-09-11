FROM golang:1.22-alpine3.20 AS builder

RUN apk add --no-cache --update git gcc rust

COPY . /src
WORKDIR /src

RUN CGO_ENABLED=0 go build -a -ldflags "-linkmode external -extldflags -static" -o /usr/local/bin/supervisord github.com/cyralinc/supervisord

FROM scratch

COPY --from=builder /usr/local/bin/supervisord /usr/local/bin/supervisord

ENTRYPOINT ["/usr/local/bin/supervisord"]
