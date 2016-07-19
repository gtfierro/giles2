#!/bin/bash

set -e


echo "ARG $1"
if [[ $1 = "bash" || $1 = "shell" ]]
then
  set +ex
  bash -i
  exit 0
fi

if [ -z "$BTRDB_SERVER" ]
then
  echo "The environment variable BTRDB_SERVER must be set"
  exit 1
fi

if [ -z "$MONGO_SERVER" ]
then
  echo "The environment variable MONGO_SERVER must be set"
  exit 1
fi

if [ -z "$GILES_BW_ENTITY" ]
then
  echo "The environment variable GILES_BW_ENTITY must be set"
  exit 1
fi

if [ -z "$GILES_BW_NAMESPACE" ]
then
  echo "The environment variable GILES_BW_NAMESPACE must be set"
  exit 1
fi

if [ -z "$GILES_BW_LISTEN" ]
then
  echo "The environment variable GILES_BW_LISTEN must be set"
  exit 1
fi

: ${GILES_HTTP_ENABLED:=true}
: ${GILES_HTTP_PORT:=8079}
: ${GILES_BOSSWAVE_ENABLED:=true}
: ${GILES_WEBSOCKET_ENABLED:=false}
: ${GILES_TCPJSON_ENABLED:=false}
: ${GILES_MSGPACKUDP_ENABLED:=false}

BTRDB_PORT=$(echo $BTRDB_SERVER | sed 's/.*[^:]\+:\([0-9]\+\).*/\1/')
BTRDB_ADDR=$(echo $BTRDB_SERVER | sed 's/\([^:]\+\):[0-9]\+.*/\1/')

MONGO_PORT=$(echo $MONGO_SERVER | sed 's/.*[^:]\+:\([0-9]\+\).*/\1/')
MONGO_ADDR=$(echo $MONGO_SERVER | sed 's/\([^:]\+\):[0-9]\+.*/\1/')


cat >giles.cfg <<EOF
[archiver]
TimeseriesStore=btrdb
Objects=mongo
MetadataStore=mongo
LogLevel=DEBUG
PeriodicReport=false

[BtrDB]
Port=${BTRDB_PORT}
Address=${BTRDB_ADDR}

[Mongo]
Port=${MONGO_PORT}
Address=${MONGO_ADDR}
UpdateInterval=10

[HTTP]
Enabled=${GILES_HTTP_ENABLED}
Port=${GILES_HTTP_PORT}

[BOSSWAVE]
Entityfile=/etc/giles/${GILES_BW_ENTITY}
Namespace=${GILES_BW_NAMESPACE}
Enabled=${GILES_BOSSWAVE_ENABLED}
ListenNS=${GILES_BW_LISTEN}
Address=172.17.0.1:28589
#Address=parent:28589

[WebSocket]
Enabled=${GILES_WEBSOCKET_ENABLED}
Port=8078

[MsgPackUDP]
Enabled=${GILES_MSGPACKUDP_ENABLED}
Port=8077

[TCPJSON]
Enabled=${GILES_TCPJSON_ENABLED}
AddPort=8001
QueryPort=8002
SubscribePort=8003

[Profile]
Enabled=false
EOF

cat giles.cfg
giles2
