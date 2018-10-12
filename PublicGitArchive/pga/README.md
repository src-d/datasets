# pga: the Public Git Archive tool

Use `pga` to list and download the repositories included in [Public Git Archive](http://pga.sourced.tech).

## Installation

There are no binary distributions available yet, but we're planning on releasing them sometime soon.
In the meanwhile you'll need to compile this tool.

1. install Go (https://golang.org/doc/install).
1. fetch the source code: `go get github.com/src-d/datasets/PublicGitArchive/pga`
1. a new binary is now avilable under `$GOPATH/bin`: `echo "$(go env GOPATH)/bin"`
1. add that binary to your `PATH` environment variable or move the binary to somewhere easier to find.
1. verify the installation went well, simply run `pga -h` and you should see some help.

## Utilization

There are two subcommands in `pga`: `list` and `get`.

### Listing repositories

When you run `pga list` two things wil happen.
First a copy of the latest index for the Public Git Archive will be downloaded and cached locally.
Then `pga` will list all the URLs for the repositories in the index.

By default only the repository URL is displayed, but you can change that with the `--format` flag:

- `--format csv` (or `-f cvs`) will print CVS rows with all the details,
- `--format json` (or `-f json`) will print do the same for JSON.

#### Filtering results

You can now add some filters to decide which ones you want to see, for now we've implemented only two
of them:

- `--lang java,go` (or `-l java,go`) will list only repositories that have at least some code in those two languages,
- `--url regexp` (or `-u regexp`) will list only the repositories for which the url matches the given regular expression.

You can always use any of your favorite tools to decide what repositories to download, such as `grep`, `jq`, or `awk` and
pass the resulting list of siva files back to `pga`.

Read below how to download repositories given the siva filenames.

### Downloading siva files

Simply replace `list` with `get`! You also get a couple of extra flags.

- `--output path` (or `-o path`) determines under what path the siva files should be stored.
  - if the path is a URL with schema `hdfs` HDFS will be used.
- `--jobs n` (or `-j n`) sets the maximum number of download hapenning concurrently, it defaults to `10`.

#### Downloading siva files given their names

Simply pass a list of siva filenames through standard input to `pga get`.

For instance, this command lists all of the repositories under github.com/src-d, filter out those with less than 50 files,
and downloads the siva files with `pga get` to the `repositories` directory.

```bash
pga list -u github.com/src-d/ -f json | jq -r 'select(.fileCount > 50) | .sivaFilenames[]' | pga get -i -o repositories
```

_Note on partial downloads_

When running `pga get` the tool will check whether the files already
downloaded match the md5 hash of the files on the server. If that's the case,
the files will not be downloaded.

This provides a simple way to resume failed downloads. Simply run the tool again.
