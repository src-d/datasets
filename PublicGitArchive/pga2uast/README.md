# pga2uast

Extract [Babelfish UASTs](https://docs.sourced.tech/babelfish/uast/uast-specification-v2) from siva files reliably and quickly.
pga2uast is a CLI application which takes a directory and parses each file in each HEAD revision in each siva file in that directory.
There are two output formats supported: `zip` and `parquet`. In any format, UASTs are serialized in Protocol Buffers,
you can use any available [Babelfish client](https://docs.sourced.tech/babelfish/using-babelfish/clients) to load them.


## Installation

Currently, you have to build from source:

```
# Install the Go compiler from https://golang.org/
go mod download
go build pga2uast.go
# You will get ./pga2uast
```

## Usage

In short,

```
./pga2uast -m -lbash,cpp,csharp,go,java,javascript,php,python,ruby /path/to/directory/with/.siva/files
```

However, there are several options, this is the output of `--help`:

```
Usage of ./pga2uast:
  -b, --bblfsh string       Babelfish server address. (default "0.0.0.0:9432")
  -f, --format string       Output format: choose one of zip, parquet. (default "zip")
  -l, --languages strings   Programming languages to parse. The full list is at https://docs.sourced.tech/babelfish/languages Several values can be specified separated by commas. The strings should be lower case. The special value "all" disables any filtering. Example: --languages=c++,python (default [all])
  -m, --monitor             Activate the advanced detection of "bad" repositories and automatic restart on failures.
  -o, --output string       Output directory where to save the results. (default "uast")
  -t, --timeout duration    Bablefish parse timeout. (default 1m0s)
  -n, --workers int         Number of goroutines to parse UASTs. (default 2x<your cpu cores>)
```

## Robustness

The "monitored" mode will relaunch the parsing from scratch if the worker process dies. The most common reason for this is the out-of-memory error.
The master process maintains "blacklist.txt" where the "bad" siva files are listed.
You should tune `-n/--workers` to stay small enough for your free RAM size to not blacklist pretty much every siva file.

## License

[MIT](https://tldrlegal.com/license/mit-license).
