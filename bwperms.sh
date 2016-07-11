#!/bin/bash

set -u

fromEntity=$1
toEntity=$2
deployNS=$3
listenNS=$4

echo "From $fromEntity"
echo "To $toEntity"
echo "Deploy On: $deployNS"
echo "Listen on: $listenNS"

bw2 bc -t $toEntity -u $deployNS/s.giles/0/i.archiver/slot/query -x C
if [ $? != 0 ]; then
    echo "Granting PC* to" $deployNS/s.giles/0/i.archiver/*
    bw2 mkdot -f $fromEntity -t $toEntity -u $deployNS/s.giles/0/i.archiver/* -x 'PC*'
fi

bw2 bc -t $toEntity -u $deployNS/s.giles/0/i.archiver/signal/+ -x C
if [ $? != 0 ]; then
    echo "Granting PC* to" $deployNS/s.giles/0/i.archiver/*
    bw2 mkdot -f $fromEntity -t $toEntity -u $deployNS/s.giles/0/i.archiver/* -x 'PC*'
fi

bw2 bc -t $toEntity -u $listenNS/*
if [ $? != 0 ]; then
    echo "Granting C* to" $listenNS/*
    bw2 mkdot -f $fromEntity -t $toEntity -u $listenNS/* -x 'C*'
fi
