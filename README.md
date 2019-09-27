# source{d} Datasets [![Build Status](https://travis-ci.com/src-d/datasets.svg?branch=master)](https://travis-ci.com/src-d/datasets) [![Build status](https://ci.appveyor.com/api/projects/status/b2en9yo9142qgadh?svg=true)](https://ci.appveyor.com/project/vmarkovtsev/datasets)

source{d} datasets for source code analysis and [machine learning on source code (ML on Code)](https://github.com/src-d/awesome-machine-learning-on-source-code).

This repository contains all the needed tools and scripts to reproduce the datasets, as well as the academic papers they may relate to.

## Available datasets

### Public Git Archive

- [Public Git Archive](PublicGitArchive)
- Size: 6TB
- Description: 260k+ top-bookmarked repositories from GitHub, consisting of 136M+ files and ~28 billion lines of code.

### Programming Language Identifiers

- [Programming Language Identifiers](Identifiers)
- Size: 1GB
- Description: ~49M distinct identifiers extracted from 10+ programming languages.

### Code duplicates

- [Manually labelled pairs of files and functions](Duplicates)
- Size: 250MB
- Description: 2k Java file and 600 Java function pairs labeled as similar or different by several programmers.

### Pull Request review comments

- [PR review comments](ReviewComments)
- Size: 1.5GB
- Description: 25.3 million GitHub PR review comments since January 2015 till December 2018.

### Commit messages

- [Commit messages](CommitMessages)
- Size: 46GB
- Description: 1.3 billion GitHub commit messages till March 2019.

### Structural commit features

- [Structural commit features](StructuralCommitFeatures)
- Size: 1.9GB
- Description: 1.6 million commits in 622 Java repositories on GitHub.

### DockerHub Metadata

- [DockerHub Metadata](DockerHubMetadata)
- Size: 1.4GB
- Description: 1.46 million Docker image configuration and manifest files on [DockerHub](https://hub.docker.com/) fetched in June 2019.

### DockerHub Packages

- [DockerHub Packages](DockerHubPackages)
- Size: 15GB
- Description: 419092 analyzed Docker images: lists of native, Python and Node packages on [DockerHub](https://hub.docker.com/) fetched in summer 2019.

### Typos
- [Typos](Typos)
- Size: 1MB
- Description: 7375 typos in source code identifier names found in GitHub repositories.

### Parallel Corpus of Code and Comments

- [The CodeSearchNet Challenge](https://arxiv.org/abs/1909.09436)
- Size: 20GB
- Description: 2M Code Comment Pairs, and 6M Total code snippets.  Benchmarks for information retrieval with a leaderboard hosted on [Weights & Biases](https://app.wandb.ai/github/CodeSearchNet/benchmark)

## Contributions

Contributions are very welcome, please see [CONTRIBUTING.md](CONTRIBUTING.md) and [code of conduct](CODE_OF_CONDUCT.md).

## License

The tools and scripts are licensed under Apache 2.0, see [LICENSE.md](LICENSE.md).
