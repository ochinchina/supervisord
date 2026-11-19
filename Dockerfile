FROM golang:1.27.3-alpine@sha256:d200d2dd76b03a99b52651fd1325aff7dcd35422c35c01bac56571b0507f3deb AS builder

COPY . /src
WORKDIR /src

RUN CGO_ENABLED=0 go build -a -tags release -ldflags "-s -w" -o /usr/local/bin/supervisord .

FROM scratch

COPY --from=builder /usr/local/bin/supervisord /usr/local/bin/supervisord

ENTRYPOINT ["/usr/local/bin/supervisord"]
