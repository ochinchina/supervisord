FROM golang:1.27.4-alpine@sha256:70bd7ddea8b6cbe5d6943cde279d6b67feb10cc924fb9100d1d8757c1575b280 AS builder

COPY . /src
WORKDIR /src

RUN CGO_ENABLED=0 go build -a -tags release -ldflags "-s -w" -o /usr/local/bin/supervisord .

FROM scratch

COPY --from=builder /usr/local/bin/supervisord /usr/local/bin/supervisord

ENTRYPOINT ["/usr/local/bin/supervisord"]
