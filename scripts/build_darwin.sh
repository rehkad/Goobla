#!/bin/sh

set -e

status() { echo >&2 ">>> $@"; }
usage() {
    echo "usage: $(basename $0) [build [sign]]"
    exit 1
}

export VERSION=${VERSION:-$(git describe --tags --first-parent --abbrev=7 --long --dirty --always | sed -e "s/^v//g")}
export GOFLAGS="'-mod=vendor -ldflags=-w -s \"-X=github.com/goobla/goobla/version.Version=${VERSION#v}\" \"-X=github.com/goobla/goobla/server.mode=release\"'"
export CGO_CPPFLAGS='-mmacosx-version-min=11.3'

ARCHS="arm64 amd64"
while getopts "a:h" OPTION; do
    case $OPTION in
        a) ARCHS=$OPTARG ;;
        h) usage ;;
    esac
done

shift $(( $OPTIND - 1 ))

_build_darwin() {
    for ARCH in $ARCHS; do
        status "Building darwin $ARCH"
        INSTALL_PREFIX=dist/darwin-$ARCH/
        GOOS=darwin GOARCH=$ARCH CGO_ENABLED=1 go build -o $INSTALL_PREFIX .

        if [ "$ARCH" = "amd64" ]; then
            status "Building darwin $ARCH dynamic backends"
            cmake -B build/darwin-$ARCH \
                -DCMAKE_OSX_ARCHITECTURES=x86_64 \
                -DCMAKE_OSX_DEPLOYMENT_TARGET=11.3 \
                -DCMAKE_INSTALL_PREFIX=$INSTALL_PREFIX
            cmake --build build/darwin-$ARCH --target ggml-cpu -j
            cmake --install build/darwin-$ARCH --component CPU
        fi
    done
}

_sign_darwin() {
    status "Creating universal binary..."
    mkdir -p dist/darwin
    lipo -create -output dist/darwin/goobla dist/darwin-*/goobla
    chmod +x dist/darwin/goobla

    if [ -n "$APPLE_IDENTITY" ]; then
        for F in dist/darwin/goobla dist/darwin-amd64/lib/goobla/*; do
            codesign -f --timestamp -s "$APPLE_IDENTITY" --identifier ai.goobla.goobla --options=runtime $F
        done

        # create a temporary zip for notarization
        TEMP=$(mktemp -u).zip
        ditto -c -k --keepParent dist/darwin/goobla "$TEMP"
        xcrun notarytool submit "$TEMP" --wait --timeout 10m --apple-id $APPLE_ID --password $APPLE_PASSWORD --team-id $APPLE_TEAM_ID
        rm -f "$TEMP"
    fi

    status "Creating universal tarball..."
    tar -cf dist/goobla-darwin.tar --strip-components 2 dist/darwin/goobla
    tar -rf dist/goobla-darwin.tar --strip-components 4 dist/darwin-amd64/lib/
    gzip -9vc <dist/goobla-darwin.tar >dist/goobla-darwin.tgz
}

_build_macapp() {
    # build and optionally sign the mac app
    npm install --prefix macapp
    if [ -n "$APPLE_IDENTITY" ]; then
        npm run --prefix macapp make:sign
    else
        npm run --prefix macapp make
    fi

    mv ./macapp/out/make/zip/darwin/universal/Goobla-darwin-universal-$VERSION.zip dist/Goobla-darwin.zip
}

if [ "$#" -eq 0 ]; then
    _build_darwin
    _sign_darwin
    _build_macapp
    exit 0
fi

for CMD in "$@"; do
    case $CMD in
        build) _build_darwin ;;
        sign) _sign_darwin ;;
        macapp) _build_macapp ;;
        *) usage ;;
    esac
done
