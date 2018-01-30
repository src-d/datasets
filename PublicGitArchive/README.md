Public Git Archive
==================

This dataset consists of two parts:

* [Siva](https://github.com/src-d/go-siva) files with Git repositories.
* Index file in CSV format.

## Tools

* [multitool](multitool) - compile the list of repositories and retrieve an existing dataset.

## Download

To download an existing dataset, execute:

```
multitool get-index -o index.txt.gz
multitool get-dataset -o /path/where/repositories/will/be/stored
```

`get-dataset` command has `-j/--workers` argument which specifies the number of downloading threads
to run.

Both `get-index` and `get-dataset` have `-b/--base` argument which specifies the base URL of the datasets.
source{d}'s address is hardcoded to be the default.

## Reproduction

#### Obtain the list of repositories to clone

The list must be a text file with one URL per line. The paper chooses
repositories on GitHub with ≥50 stars, which is equivalent to
the following commands which generate `list.txt`:

```
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
