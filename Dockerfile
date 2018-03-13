FROM golang:latest AS builder

ADD . /go/src/github.com/ochinchina/supervisord
RUN go get -v -u github.com/ochinchina/supervisord && go build -o /usr/local/bin/supervisord github.com/ochinchina/supervisord

FROM alpine:latest

COPY --from=builder /usr/local/bin/supervisord /usr/local/bin/supervisord
ENTRYPOINT ["/usr/local/bin/supervisord"]
