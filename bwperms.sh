#!/bin/bash

set -ux

if [ -z ${1+x} ]; then
    echo "Usage: ./bwperms.sh from to deployNS listenNS"
    exit 1
fi

fromEntity=$1
toEntity=$2
deployNS=$3
listenNS=$4

echo "From $fromEntity"
echo "To $toEntity"
echo "Deploy On: $deployNS"
echo "Listen on: $listenNS"

echo "Checking PC* to" $deployNS/s.giles/*
bw2 bc -t $toEntity -u $deployNS/s.giles/* -x 'PC*'
if [ $? != 0 ]; then
    echo "Granting PC* to" $deployNS/s.giles/*
    bw2 mkdot -e 5y -f $fromEntity -t $toEntity -u $deployNS/s.giles/* -x 'PC*'
fi

echo "Checking C* to" $deployNS/*
bw2 bc -t $toEntity -u $listenNS/* -x 'C*'
if [ $? != 0 ]; then
    echo "Granting C* to" $listenNS/*
    bw2 mkdot -e 5y -f $fromEntity -t $toEntity -u $listenNS/* -x 'C*'
fi
