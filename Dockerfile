FROM golang:alpine AS builder

RUN apk update && \
    apk add git 

# uncomments this if golang.org is not accessible
#RUN mkdir -p $GOPATH/src/golang.org/x && \
#    cd $GOPATH/src/golang.org/x && \
#   git clone https://github.com/golang/crypto && \
#    git clone https://github.com/golang/sys 

RUN go get -v -u github.com/ochinchina/supervisord && \
    go build -o /usr/local/bin/supervisord github.com/ochinchina/supervisord

FROM alpine:latest

COPY --from=builder /usr/local/bin/supervisord /usr/local/bin/supervisord
ENTRYPOINT ["/usr/local/bin/supervisord"]
