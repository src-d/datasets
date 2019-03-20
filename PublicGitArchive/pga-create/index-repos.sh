#!/bin/sh

# This script is intended to be used inside the docker image for 
# https://github.com/src-d/datasets/tree/master/PublicGitArchive/pga-create
# The absolut paths refers to mounted volumes inside the docker container.
# See the Dockerfile for more information.

set -e

CONFIG_ROOT_REPOSITORIES_DIR=/pga/root-repositories \
CONFIG_CLEAN_TEMP_DIR=true \
CONFIG_BUCKETSIZE=$BUCKET_SIZE \
pga-create index --debug --repos-file=/pga/data/pga.list

pga-create set-forks -f /pga/data/index.csv -o /pga/data/index_$PGA_VERSION.csv

tar -czf /pga/root-repositories/index_$PGA_VERSION.tar.gz -C /pga/data/ index_$PGA_VERSION.csv

