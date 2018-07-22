#!/bin/bash

set -e

version=$(cat /app/version.go | grep "const VERSION" | awk '{print $NF}' | sed 's/"//g')

package() {
    fpm \
        -C /app/dist/linux/ \
        -s dir \
        -t $1 \
        -n snagsby \
        -v $version \
        -p ./dist \
        ./=/usr/local/bin
}

package deb
package rpm
