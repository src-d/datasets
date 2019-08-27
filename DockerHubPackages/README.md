DockerHub Metadata ![size 15GB](https://img.shields.io/badge/size-15GB-green.svg)
==================

[Download.](https://drive.google.com/file/d/1IZ0CO-MqEWNWd3Ud3pDsX6t8tPslY0Xu)

419092 lists of native, Python and Node packages installed in Docker images on [DockerHub](https://hub.docker.com/) fetched in summer 2019.
The collection and analysis of this dataset was the internship project of @glimow. We expect further improvements in the future.

The dataset consists of a single archive: [`packages.tar.xz`](https://drive.google.com/file/d/1IZ0CO-MqEWNWd3Ud3pDsX6t8tPslY0Xu),
320MB download size, 15GB uncompressed.
There are 419092 JSON files spread over prefix directories inside.

### Format

For each Docker image, the top level directory is the first two letters of the image name, the inner directories correspond to the name parts after splitting by `/`.
`:latest` is stripped from the file names.
Examples: the packages for `tensorflow/tensorflow:2.0.0b0` will be at
`te/tensorflow/tensorflow:2.0.0b0.json`, and for `mongo:latest` at `mo/mongo.json`.

The JSON schema is

```
{
    "image": "mongo:latest",
    "size": 384788,  // overall size in KB
    "distribution": "ubuntu",  // options: ubuntu, debian, alpine, raspbian, centos, fedora, amzn, arch, ol
    "version": "16.04",
    "packages": {
        "native": [
            [
                "adduser",
                "3.113+nmu3ubuntu4",
                664000  // size in bytes
            ],
            ...
        ],
        "python3": [
            [
                "pip",
                "19.0.3",
                7788  // size in KB
            ],
            ...
        ],
        "node": [
            [
                "express",
                "4.16.3",
                256  // size in KB
            ],
            ...
        ]
    }
}
```

### Use cases

There is plenty of use cases, e.g.

* Attention-based NN to predict native packages from Python/Node packages. Never mess with missing headers and `*-dev` packages anymore!
* Library embeddings and classification.
* Redundant packages prediction.
* Frequent itemsets of various kind.
* Traditional statistics.

See the [`research`](research) directory for inspiration. We are preparing a blog post about it.
5% of Python packages belonging to the same version have different sizes! 🤯

### Origin

We took the [DockerHub metadata dataset](../DockerHubMetadata), looked for mentions of Python and Node in the layer configurations,
then seriated them according to the layer edit distance to ensure the optimal pull performance and finally pulled them one by one on
multiple machines in parallel. Each pulled image was analyzed and the results saved to a JSON file. See [`code`](code) for the reproduction source code.

### Limitations

* Some images failed to pull.
* Empty JSON files mean analysis failures.
* Python and Node packages are tricky to find. The current detection algorithm can be improved much. For example, we did not seek for `virtualenv`s (why on earth you want it inside a Docker image??).
* We ignored Python 2.
* We only handled specific Linux distributions.

### License

Code: [MIT](https://choosealicense.com/licenses/mit/).
Compilation: [Open Data Commons Open Database License (ODbL)](https://opendatacommons.org/licenses/odbl/).
Actual contents: [DockerHub Terms of Service](https://www.docker.com/legal/docker-terms-service).
