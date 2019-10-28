# pga: the Public Git Archive tool

Use `pga` to list and download the repositories included in [Public Git Archive](http://pga.sourced.tech).

## Installation

There are no binary distributions available yet, but we're planning on releasing them sometime soon.
In the meanwhile you'll need to compile this tool.

1. install Go 1.11+ (https://golang.org/doc/install) and `export GO111MODULE=on`.
1. fetch and build: `go get github.com/src-d/datasets/PublicGitArchive/pga`
1. add the built binary `pga` to your `PATH` environment variable or move it to somewhere easier to find.
1. verify the installation went well, simply run `pga -h` and you should see some help.

## Utilization

There are three subcommands in `pga`: `list`, `get`, and `siva`.

### Datasets

Two datasets are exposed through this tool, and both the `list` and `get` command can be used to explore and retrieve them. To do so, you must specify with a keyword which dataset you want to work on :

- `siva`: The original Public Git Archive dataset, made up of Siva files.
- `uast`: The [dataset](../../PublicGitArchiveUASTs) created by extracting UASTs from the HEAD commit of each repository, made up of Parquet files.

Note that the `siva` _command_ does not work on Parquet files.

### Listing repositories

When you run `pga list` two things wil happen.
First a copy of the latest index for the dataset specified will be downloaded and cached locally.
Then `pga` will list all the URLs for the repositories in the index.

By default only the repository URL is displayed, but you can change that with the `--format` flag:

- `--format csv` (or `-f cvs`) will print CVS rows with all the details,
- `--format json` (or `-f json`) will print do the same for JSON.

The extended information includes the fields:
- `URL`, `SIVA_FILENAMES`, `FILE_COUNT`, `LANGS`,`LANGS_BYTE_COUNT`, `LANGS_LINES_COUNT`,`LANGS_FILES_COUNT`, `COMMITS_COUNT`, `BRANCHES_COUNT`, `FORK_COUNT`, `EMPTY_LINES_COUNT`, `CODE_LINES_COUNT`, `COMMENT_LINES_COUNT`, `LICENSE`, `STARS` and `SIZE` for the original dataset.
- `URL`, `PARQUET_FILENAMES`, `FILE_COUNT`, `SIZE`, `FILE_EXTRACT_RATE`, `BYTE_EXTRACT_RATE`, `LANGS`, `LANGS_FILE_COUNT`, `LANGS_BYTE_COUNT`, `LANGS_FILE_EXTRACT_RATE` and `LANGS_BYTE_EXTRACT_RATE` for the UASTs dataset.

Note that the fields `STARS` and `SIZE` can hold the value `-1` to point out that the index doesn't have information about those for the orginal dataset. This ensures compatibility between different index versions.

`SIZE` represents the sum of the sizes of all the siva files you need to collect to get the complete repository. Because a siva file can hold several repositories information, when you need to download more than one repository the total amount of bytes to be downloaded will be at most the sum of their `SIZES` values though it could be less if they share any of the siva files.

#### Filtering results

You can now add some filters to decide which ones you want to see, for now we've implemented only two
of them:

- `--lang java,go` (or `-l java,go`) will list only repositories that have at least some code in those two languages,
- `--url regexp` (or `-u regexp`) will list only the repositories for which the url matches the given regular expression.

You can always use any of your favorite tools to decide what repositories to download, such as `grep`, `jq`, or `awk` and
pass the resulting list of siva files back to `pga`.

Read below how to download repositories given the siva filenames.

### Downloading files

Simply replace `list` with `get`! You also get a couple of extra flags.

- `--output path` (or `-o path`) determines under what path the siva files should be stored.
  - if the path is a URL with schema `hdfs` HDFS will be used.
- `--jobs n` (or `-j n`) sets the maximum number of download hapenning concurrently, it defaults to `10`.

#### Downloading files given their names

Simply pass a list of siva or parquet filenames through standard input to `pga get`.

For instance, this command lists all of the repositories under github.com/src-d, filter out those with less than 50 files,
and downloads the siva files with `pga get` to the `repositories` directory.

```bash
pga list siva -u github.com/src-d/ -f json | jq -r 'select(.fileCount > 50) | .sivaFilenames[]' | pga get siva -i -o repositories
```

_Note on partial downloads_

When running `pga get` the tool will check whether the files already
downloaded match the md5 hash of the files on the server. If that's the case,
the files will not be downloaded.

This provides a simple way to resume failed downloads. Simply run the tool again.

### Extracting files from downloaded siva-s

The following will write the contents of each HEAD revision contained in a siva file to the current
working directory:

```bash
pga siva dump /path/to/siva
```

`-o /output/path` allows setting the output path other than the current working directory.

### Listing the commits and references in a downloaded siva file

```bash
pga siva list /path/to/siva
```

The output format is JSON. In the `"commits"` dictionary, each value is the list of the commit's parents.
In the `"references"` dictionary, each value is the reference's target.

### Dumping the raw siva contents (advanced)

It is possible to extract the raw contents of a siva archive with

```bash
pga siva unpack /path/to/siva
```

It is possible to specify a regular expression for matching specific files to be extracted: `-m/--match`.
