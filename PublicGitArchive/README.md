Public Git Archive ![size 6.0TB](https://img.shields.io/badge/size-6.0TB-green.svg)
==================

[Paper](https://arxiv.org/abs/1803.10144) (accepted to [MSR'18](https://2018.msrconf.org/track/msr-2018-Data-Showcase-Papers)). [Presentation](http://vmarkovtsev.github.io/msr-2018-gothenburg/).

This dataset consists of two parts:

* [Siva](https://github.com/src-d/go-siva) files with Git repositories.
* Index file in CSV format.

## Tools

* [pga](pga) - explore the dataset, or download its contents easily.
* [pga-create](pga-create) - reproduce PGA dataset generation.
* [borges-indexer](borges-indexer) - exports a CSV file with metadata from repositories fetched with Borges.

## Listing and downloading

To see the full list of repositories in the dataset or download it, you will need to install
[pga](pga).
Simply install Go and then run `go get github.com/src-d/datasets/PublicGitArchive/pga`.

Then to list all of the repositories in the dataset, simply run:

```bash
pga list
```

If you'd rather get a detailed dump of the dataset (not including the file contents)
you can choose either `pga list -f json` or `pga list -f csv`.

To download the full dataset, execute:

```bash
pga get
```

Or if you want to download only those repositories containing at least a line of Java code:

```bash
pga get -l java
```

The `pga` command has `-j/--workers` argument which specifies the number of downloading threads to run, it defaults to 10.

For more information, check the [pga documentation](pga), or simply run `pga -h`.

## Reproduction

Refer to [pga-create](pga-create) documentation for more details about how PGA is generated.

## Blacklist

We understand that some GitHub projects may become private or deleted with time. Previous dataset snapshots will continue to include such dead code. If you are the author and want to remove your project from all present and future public snapshots, please send a request to `datasets@sourced.tech`.
