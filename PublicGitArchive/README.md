Public Git Archive ![size 3.0TB](https://img.shields.io/badge/size-3.0TB-green.svg)
==================

[Paper](https://arxiv.org/abs/1803.10144) (accepted to [MSR'18](https://2018.msrconf.org/track/msr-2018-Data-Showcase-Papers)).

This dataset consists of two parts:

* [Siva](https://github.com/src-d/go-siva) files with Git repositories.
* Index file in CSV format.

## Tools

* [pga](pga) - explore the dataset, or download its contents easily.
* [multitool](multitool) - compile the list of repositories and retrieve an existing dataset.
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

#### Obtain the list of repositories to clone

The list must be a text file with one URL per line. The paper chooses
repositories on GitHub with ≥50 stars, which is equivalent to
the following commands which generate `list.txt`:

```bash
multitool discover -s stars.txt -r repos.txt.gz
multitool select -s stars.txt -r repos.txt.gz -m 50 > list.txt
```

#### Cloning repositories

You are going to need [Borges](https://github.com/src-d/borges) and all it's
dependencies: RabbitMQ and PostgreSQL. The following commands are an artificial
simplified cloning scenario, please refer to Borges docs for the detailed manual.

In the first terminal execute

```
borges init
borges producer --source=file --file list.txt
```

In the second terminal execute

```
export CONFIG_ROOT_REPOSITORIES_DIR=/path/where/repositories/will/be/stored
borges consumer
```

#### Processing repositories

To process the downloaded repositories you will need `borges-indexer` tool included in this project, and run it querying the database populated in the previous step. This will generate a CSV with the extracted information of all those repositories.

Same environment variables as in borges can be used to configure the database access.

```
cd borges-indexer
make build
./borges-indexer -debug -logfile=borges-indexer.log
```

The arguments accepted by borges indexer are the following:
* `-debug`: print more verbose logs that can be used for debugging purposes
* `-logfile=<LOGFILE PATH>`: path to the file where logs will be written
* `-limit=N`: max number of repositories to process (useful for batch processing)
* `-offset=N`: skip the first N repositories (useful for batch processing)

**NOTE:** this spawns as many workers as CPUs are available in the machine. Take into account that some repositories may be considerably large and this process may take a very big amount of memory in the machine.

After being processed with `borges-indexer` you will have a `result.csv` file with all the content you need. The only missing content will be the `FORK_COUNT`, but for that you can use the also included `set-forks` command.

```
./set-forks
```

This will take `result.csv` and add the forks to it, resulting in a `result_forks.csv` file with the same data you had in the original CSV, only with the forks added.

## Blacklist

We understand that some GitHub projects may become private or deleted with time. Previous dataset snapshots will continue to include such dead code. If you are the author and want to remove your project from all present and future public snapshots, please send a request to `datasets@sourced.tech`.
