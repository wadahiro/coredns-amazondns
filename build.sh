#!/bin/sh

VERSION=0.1
TAG=master

if [ "$1" = "" ]; then
    docker run --rm \
        -v $(pwd):/go/src/github.com/wadahiro/coredns-amazondns \
        -v $(pwd)/.tmp:/go \
        -w /go/src/github.com/wadahiro/coredns-amazondns \
        golang:1.9 ./build.sh $TAG
else 
    echo "Building CoreDNS:$1 with amazondns..."

    go get github.com/coredns/coredns
    cd /go/src/github.com/coredns/coredns

    git reset --hard HEAD
    git clean -f

    git checkout $1
    if [ "$?" -ne 0 ]; then
        echo "Invalid tag: $1"
        exit 1
    fi

    sed -i -e "/^route53:route53$/i amazondns:github.com/wadahiro/coredns-amazondns" plugin.cfg 

    if [ "$?" -ne 0 ]; then
        echo "Failed"
        exit 1
    fi

    cat plugin.cfg

    go generate
    #make
    go build

    cp coredns /go/src/github.com/wadahiro/coredns-amazondns/
    tar cvzf coredns-amazondns-$VERSION.tar.gz coredns
    mv coredns-*.tar.gz /go/src/github.com/wadahiro/coredns-amazondns/
fi

