#!/bin/bash

set -e

dist_dir=$(pwd)/dist
mkdir -p $dist_dir

version=$VERSION

for os in linux darwin; do
    for arch in arm64 amd64; do
        name="snagsby-$version.$os-$arch"
        path="$dist_dir/$name"
        echo "Building $name - $version"
        GOOS=$os GOARCH=$arch CGO_ENABLED=0 go build \
            -ldflags "$GO_LDFLAGS" \
            -o $path
        gzip < $path > $path.gz
        cp $path $dist_dir/snagsby
        (cd $dist_dir && tar zcf $path.tar.gz snagsby && rm snagsby)
        mkdir -p $dist_dir/$os
        cp $path $dist_dir/$os/snagsby
    done
done
