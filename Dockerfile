FROM golang:alpine AS builder

RUN apk add --no-cache --update git

RUN go get -v -u github.com/ochinchina/supervisord

RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags "-linkmode external -extldflags -static" -o /usr/local/bin/supervisord github.com/ochinchina/supervisord

FROM scratch

COPY --from=builder /usr/local/bin/supervisord /usr/local/bin/supervisord

ENTRYPOINT ["/usr/local/bin/supervisord"]
