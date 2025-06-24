#!/bin/sh

set -eu

export VERSION=${VERSION:-0.0.0}
export GOFLAGS="'-mod=vendor -ldflags=-w -s \"-X=github.com/goobla/goobla/version.Version=$VERSION\" \"-X=github.com/goobla/goobla/server.mode=release\"'"

docker build \
    --push \
    --platform=linux/arm64,linux/amd64 \
    --build-arg=VERSION \
    --build-arg=GOFLAGS \
    -f Dockerfile \
    -t goobla/goobla -t goobla/goobla:$VERSION \
    .
