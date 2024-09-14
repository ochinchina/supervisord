# Distributed under the terms of the Modified BSD License.

ARG BASE_NAMESPACE
ARG BUILD_IMG="golang:latest"
ARG BASE_IMG="ubuntu:latest"

FROM ${BASE_NAMESPACE:+$BASE_NAMESPACE/}${BUILD_IMG} AS builder

RUN if command -v apt-get >/dev/null 2>&1; then \
        echo "Detected Debian/Ubuntu environment. Installing packages using apt-get..." ; \
        apt-get update && apt-get install -y gcc ; \
    elif command -v apk >/dev/null 2>&1; then \
        echo "Detected Alpine environment. Installing packages using apk..." ; \
        apk add --no-cache gcc musl-dev ; \
    else \
        echo "Unsupported environment. Neither apt-get nor apk found." ; \
        return 1 ; \
    fi

COPY    ./src /tmp/src
WORKDIR       /tmp/src
RUN set -eux && pwd && ls -alh \
 && mkdir -pv /opt/supervisord && mv webgui etc /opt/supervisord/ \
 && cd supervisord \
 && go mod tidy \
 && CGO_ENABLED=1 go build -a -ldflags "-linkmode external -extldflags -static" -o /opt/supervisord/ \
 && /opt/supervisord/supervisord version


ARG BASE_IMG="atom"
FROM ${BASE_IMG}
LABEL maintainer="haobibo@gmail.com"
COPY --from=builder /opt/supervisord /opt/supervisord
EXPOSE 9001
WORKDIR /opt/supervisord/
CMD ["/opt/supervisord/supervisord", "-c", "etc/supervisor.conf"]
