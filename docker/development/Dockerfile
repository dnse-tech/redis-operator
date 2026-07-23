FROM golang:1.25.12-alpine

RUN apk --no-cache add \
    bash \
    git \
    g++ \
    openssl

# Code generation runs via the pinned CODEGEN_IMAGE (see Makefile), not an
# embedded copy, so no in-image code-generator is fetched here.

# Mock creator. mockery ships assets as mockery_<ver>_Linux_<arch>.tar.gz with
# arch in Go form (arm64/x86_64); alpine's busybox `uname -m` returns aarch64,
# so map it before building the URL.
ARG MOCKERY_VERSION="2.9.6"
RUN ARCH="$(uname -m)"; case "$ARCH" in aarch64) ARCH=arm64 ;; esac; \
    wget -c "https://github.com/vektra/mockery/releases/download/v${MOCKERY_VERSION}/mockery_${MOCKERY_VERSION}_Linux_${ARCH}.tar.gz" -O - | tar -xz -C /go/bin/

# Create user
ARG uid=1000
ARG gid=1000
RUN addgroup -g $gid rf && \
    adduser -D -u $uid -G rf rf && \
    chown rf:rf -R /go


USER rf
WORKDIR /go/src/github.com/spotahome/redis-operator
