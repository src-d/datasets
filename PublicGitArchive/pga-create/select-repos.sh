#!/bin/sh

# This script is intended to be used inside the docker image for 
# https://github.com/src-d/datasets/tree/master/PublicGitArchive/pga-create
# The absolut paths refers to mounted volumes inside the docker container.
# See the Dockerfile for more information.

set -e

pga-create discover
pga-create select -m $STARS >/pga/data/pga.list

