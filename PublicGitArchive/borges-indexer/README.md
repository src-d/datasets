# borges-indexer
Exports a CSV index file from repositories fetched with Borges.

## Installation

```
go get github.com/src-d/datasets/PublicGitArchive/borges-indexer
```

## Processing repositories

To process repositories you will need to have previously downloaded repositories with [borges](https://github.com/src-d/borges) and have a PostgreSQL database populated with data (which is automatically done by borges). This will generate a CSV with the extracted information of all those repositories.

Same environment variables as in borges can be used to configure the database access.

```
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