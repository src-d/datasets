# pga-create

Tool to create the PGA dataset.

The following commands exist:

* `discover` - extract the needed information from GHTorrent MySQL dump on the fly. Requires only 1.5 GB of storage.
* `select` - compile the list of repositories to clone according to various filters, such as stars or languages.
* `index` - create the index
* `set-forks` - add fork counts

## Installation

There are 64-bit binaries for Linux, MacOS and Windows on [Releases page](https://github.com/src-d/datasets/releases).

## Build from source

```
go get -v github.com/src-d/datasets/PublicGitArchive/pga-create
```

### Obtain the list of repositories to clone

The list must be a text file with one URL per line. The paper chooses
repositories on GitHub with ≥50 stars, which is equivalent to
the following commands which generate `list.txt`:

```bash
pga-create discover -s stars.txt -r repos.txt.gz
pga-create select -s stars.txt -r repos.txt.gz -m 50 > list.txt
```

### Cloning repositories

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

### Processing repositories

To process the downloaded repositories you will need the `pga-create index` command, and run it querying the database populated in the previous step. This will generate a CSV with the extracted information of all those repositories.

Same environment variables as in borges can be used to configure the database access.

```
pga-create index -debug -logfile=borges-indexer.log
```

The arguments accepted by borges indexer are the following:
* `-debug`: print more verbose logs that can be used for debugging purposes
* `-logfile=<LOGFILE PATH>`: path to the file where logs will be written
* `-limit=N`: max number of repositories to process (useful for batch processing)
* `-offset=N`: skip the first N repositories (useful for batch processing)

**NOTE:** this spawns as many workers as CPUs are available in the machine. Take into account that some repositories may be considerably large and this process may take a very big amount of memory in the machine.

After being processed with `index` you will have a `result.csv` file with all the content you need. The only missing content will be the `FORK_COUNT`, but for that you can use the also included `set-forks` command.

```
pga-create set-forks
```

This will take `result.csv` and add the forks to it, resulting in a `result_forks.csv` file with the same data you had in the original CSV, only with the forks added.

