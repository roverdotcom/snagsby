#!/bin/bash

set -e

version=$VERSION

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
