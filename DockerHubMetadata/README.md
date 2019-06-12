DockerHub Metadata ![size 1.4GB](https://img.shields.io/badge/size-1.4GB-green.svg)
==================

[Download the configurations.](https://drive.google.com/open?id=1ZSmBR1xB9J79-Xk9gnYx3VGVp-6o0GQu)

[Download the manifests.](https://drive.google.com/open?id=1v9XcaGU71yfFrN4zKPRQ7CwGPP6rVNo2)

1.46 million Docker image configuration and manifest files on [DockerHub](https://hub.docker.com/) fetched in June 2019.
A manifest points to the layers of an image and its configuration. A configuration carries all the
metadata: architecture, OS, environment variables, entry point, default command, etc., including the layer creation history.
The latter allows to reconstruct [`docker history`](https://docs.docker.com/engine/reference/commandline/history/)
without having to pull images. As a whole, the provided information can be used to recover Dockerfile-s
for any image on DockerHub which has it.

The dataset consists of 2 files:

1. [`configs.tar.xz`](https://drive.google.com/open?id=1ZSmBR1xB9J79-Xk9gnYx3VGVp-6o0GQu) - configuration JSON files, 16GB uncompressed.
2. [`manifests.tar.xz`](https://drive.google.com/open?id=1v9XcaGU71yfFrN4zKPRQ7CwGPP6rVNo2) - manifest JSON files, 8.5GB uncompressed.

### Format

The directory structure is the same for configurations and manifests. The top level directory is
the first two letters of the image name, the inner directories correspond to the name, including the `/`.
`:latest` is stripped from the file names.
Examples: the configuration for `tensorflow/tensorflow:2.0.0b0` will be at
`te/tensorflow/tensorflow:2.0.0b0.json`, and for `mongo:latest` at `mo/mongo.json`.

The manifest format is defined at https://docs.docker.com/registry/spec/manifest-v2-2
The configuration format is defined at https://github.com/moby/moby/blob/master/image/spec/v1.2.md

### Origin

DockerHub API. We modified [skopeo](https://github.com/containers/skopeo) to fetch configurations
and manifests at blazing speed (less than 3 hours for the whole DockerHub), the modified source for
`cmd/skopeo/inspect.go` is included into this repository. Image list fetcher is written in Python
an is also included.
How to reproduce:

```
TODO: call to Tristan's script to write the list to images.txt
cp inspect.go /path/to/skopeo/cmd/skopeo/inspect.go
make -C /path/to/skopeo/ binary
cat images.txt | /path/to/skopeo/skopeo inspect
```

### Limitations

* Only i386, amd64, arm and arm64 Linux images were considered.
* Custom image registries were not processed, e.g. [microsoft-dotnet-core-samples](https://hub.docker.com/_/microsoft-dotnet-core-samples/).

### License

Code: [MIT](https://choosealicense.com/licenses/mit/).
Compilation: [Open Data Commons Open Database License (ODbL)](https://opendatacommons.org/licenses/odbl/).
Actual contents: [DockerHub Terms of Service](https://www.docker.com/legal/docker-terms-service).
