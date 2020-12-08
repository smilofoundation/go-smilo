#!/bin/bash -e
go get github.com/mitchellh/gox
go get github.com/tcnksm/ghr

export APPNAME="nats-streaming-server"
export OSARCH="linux/386 linux/amd64 linux/arm64 darwin/amd64 windows/386 windows/amd64"
export DIRS="linux-386 linux-amd64 linux-arm6 linux-arm7 linux-arm64 darwin-amd64 windows-386 windows-amd64"
export OUTDIR="pkg"

# If we have an arg, assume its a version tag and rename as appropriate.
if [[ -n $1 ]]; then
    export APPNAME=$APPNAME-$1
fi

# Build all from OSARCH list
env CGO_ENABLED=0 gox -osarch="$OSARCH" -ldflags="-s -w" -output "$OUTDIR/$APPNAME-{{.OS}}-{{.Arch}}/nats-streaming-server"

# Be explicit about the ARM builds
# ARMv6
env CGO_ENABLED=0 GOARM=6 gox -osarch="linux/arm" -ldflags="-s -w" -output "$OUTDIR/$APPNAME-linux-arm6/nats-streaming-server"
# ARMv7
env CGO_ENABLED=0 GOARM=7 gox -osarch="linux/arm" -ldflags="-s -w" -output "$OUTDIR/$APPNAME-linux-arm7/nats-streaming-server"


for dir in $DIRS; do \
    (cp README.md $OUTDIR/$APPNAME-$dir/README.md) ;\
    (cp LICENSE $OUTDIR/$APPNAME-$dir/LICENSE) ;\
    (cd $OUTDIR && zip -q $APPNAME-$dir.zip -r $APPNAME-$dir) ;\
    echo "make $OUTDIR/$APPNAME-$dir.zip" ;\
done
