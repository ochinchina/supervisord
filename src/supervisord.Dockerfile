FROM golang:alpine AS base
RUN apk add --no-cache --update git gcc rust


FROM base AS builder
COPY . /tmp/src
WORKDIR /tmp/src
RUN set -eux && pwd && ls -alh \
 && mkdir -pv /opt/supervisord && mv webgui etc /opt/supervisord/ \
 && cd supervisord \
 && go mod tidy \
 && CGO_ENABLED=1 go build -a -ldflags "-linkmode external -extldflags -static" -o /opt/supervisord/ \
 && /opt/supervisord/supervisord version


FROM busybox
COPY --from=builder /opt/supervisord /opt/supervisord
EXPOSE 9001
WORKDIR /opt/supervisord/
CMD ["/opt/supervisord/supervisord", "-c", "etc/supervisor.conf"]
