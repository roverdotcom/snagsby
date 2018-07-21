#!/bin/bash

set -e

DIR=$(pwd)/dist
mkdir -p $DIR

version=$(cat ./version.go | grep "const VERSION" | awk '{print $NF}' | sed 's/"//g')
os=$(go env GOOS)
arch=$(go env GOARCH)

for os in linux darwin; do
    name="snagsby-$version.$os-$arch"
    path="$DIR/$name"
    echo "Building $name"
    GOOS=$os GOARCH=$arch CGO_ENABLED=0 go build -o $path
    gzip < $path > $path.gz
    cp $path $DIR/snagsby
    (cd $DIR && tar zcf $path.tar.gz snagsby && rm snagsby)
done
