#!/bin/sh

# This script is intended to be used inside the docker image for 
# https://github.com/src-d/datasets/tree/master/PublicGitArchive/pga-create
# The absolut paths refers to mounted volumes inside the docker container.
# See the Dockerfile for more information.

set -e

MYSQL_DUMP=$(wget -qO - http://ghtorrent-downloads.ewi.tudelft.nl/mysql | grep "tar.gz" | tail -n 1 | cut -d '"' -f 2)
readonly PGA_DATA_PATH=/pga/data
readonly REPACK_FILE=repack-$MYSQL_DUMP
readonly PGA_LIST_FILE=pga.list

if [ ! -d $PGA_DATA_PATH ]; then
    mkdir $PGA_DATA_PATH
fi

if [ ! -f $PGA_DATA_PATH/$MYSQL_DUMP ]; then
    wget -P $PGA_DATA_PATH "http://ghtorrent-downloads.ewi.tudelft.nl/mysql/$MYSQL_DUMP"
fi

if [ ! -f $PGA_DATA_PATH/$REPACK_FILE ]; then 
    cat $PGA_DATA_PATH/$MYSQL_DUMP | pga-create repack --stdin -o $PGA_DATA_PATH/$REPACK_FILE
fi

cat $PGA_DATA_PATH/$REPACK_FILE | pga-create discover --stdin
pga-create select -m $STARS >$PGA_DATA_PATH/$PGA_LIST_FILE
