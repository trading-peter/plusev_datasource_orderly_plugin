#!/bin/bash

# Build the plugin. Pass "dev" as the first argument (or set DEV=1) to also
# build the sandbox-targeting dev variant: -tags dev adds the loopback
# network targets and the (dev) name marker (ORD-701) and writes
# plugin-dev.wasm next to the normal artifact.

DEV_MODE=0
if [ "${1:-}" = "dev" ] || [ "${DEV:-}" = "1" ]; then
    DEV_MODE=1
fi

echo "Building plugin..."
go mod tidy
GOOS=wasip1 GOARCH=wasm go build -o plugin.wasm -buildmode=c-shared .

if [ $? -eq 0 ]; then
    echo "Plugin built successfully"
else
    echo "Build failed"
    exit 1
fi

if [ "$DEV_MODE" -eq 1 ]; then
    echo "Building dev plugin (loopback targets)..."
    GOOS=wasip1 GOARCH=wasm go build -tags dev -o plugin-dev.wasm -buildmode=c-shared .

    if [ $? -eq 0 ]; then
        echo "Dev plugin built successfully"
    else
        echo "Dev build failed"
        exit 1
    fi
fi
