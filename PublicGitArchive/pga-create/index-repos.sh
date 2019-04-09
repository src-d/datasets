#!/bin/sh

# This script is intended to be used inside the docker image for 
# https://github.com/src-d/datasets/tree/master/PublicGitArchive/pga-create
# The absolut paths refers to mounted volumes inside the docker container.
# See the Dockerfile for more information.

set -e

readonly PGA_DATA_PATH=/pga/data
readonly PGA_LIST=repositories-index.csv.gz

CONFIG_ROOT_REPOSITORIES_DIR=/pga/root-repositories \
CONFIG_CLEAN_TEMP_DIR=true \
CONFIG_BUCKETSIZE=$BUCKET_SIZE \
pga-create index --debug -r $PGA_DATA_PATH/$PGA_LIST

pga-create set-forks -f $PGA_DATA_PATH/index.csv -o $PGA_DATA_PATH/index_$PGA_VERSION.csv

tar -czf /pga/root-repositories/index_$PGA_VERSION.tar.gz -C $PGA_DATA_PATH/ index_$PGA_VERSION.csv

