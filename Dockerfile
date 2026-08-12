FROM golang:1.25-alpine AS builder

COPY . /src
WORKDIR /src

RUN CGO_ENABLED=0 go build -a -tags release -ldflags "-s -w" -o /usr/local/bin/supervisord .

FROM scratch

COPY --from=builder /usr/local/bin/supervisord /usr/local/bin/supervisord

ENTRYPOINT ["/usr/local/bin/supervisord"]
