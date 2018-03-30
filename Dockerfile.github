# Use this file when golang.org/go.googlesource.com is blocked.
#
# Build with:
#
# docker build . -f Dockerfile.github -t ochinchina/supervisord:latest
#
FROM golang:alpine as builder

RUN apk add --no-cache --update git

RUN mkdir -p $GOPATH/src/golang.org/x && \
    cd $GOPATH/src/golang.org/x && \
    git clone https://github.com/golang/crypto && \
    git clone https://github.com/golang/sys

# Exit 0 to ignore meta tag complaints
RUN go get -v -u github.com/ochinchina/supervisord; exit 0

RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags "-extldflags -static" -o /usr/local/bin/supervisord github.com/ochinchina/supervisord

FROM scratch

COPY --from=builder /usr/local/bin/supervisord /usr/local/bin/supervisord

ENTRYPOINT ["/usr/local/bin/supervisord"]
