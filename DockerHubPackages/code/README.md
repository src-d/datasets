# Docker Image Analysis

Source{d} mining software for dockerhub dataset

This repo contains scripts and utilities used to produce a docker images libraries dataset by analyzing in-depth each image's filesystem.

Requirements:

- Have [Docker Installed](https://docs.docker.com/install/)
- Have [IPython Installed](https://ipython.org/install.html)

## Usage

```bash
ipython main.py ./images.txt ./packages
```

Where `./images.txt` contains the list of images to analyse, one per line. If no tag is specified, `latest` will be used.

Example `images.txt`:

```text
amancevice/superset
ubuntu:18.04
express-gateway
alpine/node
archmageinc/node-web-dev
```

And `./packages` the folder where the result will be written on disk.

The output directory structure is the same as the [DockerhubMetadata dataset](https://github.com/src-d/datasets/tree/master/DockerHubMetadata). The top level directory is the first two letters of the image name, the inner directories correspond to the name, including the /. :latest is stripped from the file names. Examples: the configuration for tensorflow/tensorflow:2.0.0b0 will be at te/tensorflow/tensorflow:2.0.0b0.json, and for mongo:latest at mo/mongo.json.

## Notes

- `show_count` is a bash script that specifically shows the amount of already fetched images in source{d}'s `typos{1-4}` nodes. It is of no use outside of the organization and should be removed before making the repo public. It is left here as documentation about the ongoing tasks.
