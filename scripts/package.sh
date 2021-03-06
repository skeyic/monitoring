#!/bin/bash
set -ex
module="monitoring"
bin="monitoring"

# GO Build
echo "build app: $module"
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags "-s" -o bin/${bin} .

echo "build docker image: $module"

docker build . -t $module:"$(date '+%Y-%m-%d-%H-%M-%S')"

rm -rf bin