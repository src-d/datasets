# pga-create

Tool to create the PGA dataset.

The following commands exist:

* `repack` - downloads latest GHTorrent MySQL dump and repacks it only with the required files (optional step).
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
pga-create discover
pga-create select -m 50 > repository_list.txt
```

### Cloning repositories

You are going to need [Borges](https://github.com/src-d/borges) and all it's
dependencies: RabbitMQ and PostgreSQL. The following commands are an artificial
simplified cloning scenario, please refer to Borges docs for the detailed manual.

In the first terminal execute

```
borges init
borges producer --source=file --file repository_list.txt
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
pga-create index --debug --logfile=pga-create-index.log
```

The options accepted by `pga-create index` are the following:
```
-o, --output=   csv file path with the results (default: data/index.csv)
--debug         show debug logs
--logfile=      write logs to file
--limit=        max number of repositories to process
--offset=       skip initial n repositories
--workers=      number of workers to use (defaults to number of CPUs)
--repos-file=   path to a file with a repository per line, only those will be processed
-s, --stars=    input path for the file with the numbers of stars per repository (default: data/stars.gz)
-r, --repositories= input path for the gzipped file with the repository names and identifiers (default: data/repositories.gz)
```

To set the `SIZE` field properly, it relies on the default temporary directories configuration for the [core-retrieval](https://github.com/src-d/core-retrieval) dependency but for the `CONFIG_CLEAN_TEMP_DIR` environment variable which must be set to `true`:

```
CONFIG_CLEAN_TEMP_DIR=true pga-create index --debug --logfile=pga-create-index.log
```

**NOTE:** this spawns as many workers as CPUs are available in the machine. Take into account that some repositories may be considerably large and this process may take a very big amount of memory in the machine.

After being processed with `index` you will have a `result.csv` file with all the content you need. The only missing content will be the `FORK_COUNT`, but for that you can use the also included `set-forks` command.

```
pga-create set-forks
```

This will take `result.csv` and add the forks to it, resulting in a `result_forks.csv` file with the same data you had in the original CSV, only with the forks added.
