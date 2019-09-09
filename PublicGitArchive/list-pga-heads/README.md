# list-pga-heads

Simple app to massively list HEADs in siva files. It can work with PGA.

## Installation

Currently, you have to build from source:

```
# Install the Go compiler from https://golang.org/
go mod download
go build list_heads.go
# You will get ./list_heads
```

## Usage

In short,

```
./list_heads -lbash,cpp,csharp,go,java,javascript,php,python,ruby /path/to/directory/with/.siva/files
```

However, there are several options, this is the output of `--help`:

```
Usage of ./list_heads:
  -f, --format string       Output format: choose one of zip, parquet. (default "zip")
  -l, --languages strings   Programming languages to parse. The full list is at https://docs.sourced.tech/babelfish/languages Several values can be specified separated by commas. The strings should be lower case. The special value "all" disables any filtering. Example: --languages=c++,python (default [all])
  -o, --output string       Output directory where to save the results. (default "files")
  -n, --workers int         Number of goroutines to read siva files. (default 16)
```

For each HEAD inside each siva file, the tool writes the list of file paths. If the format is "zip",
each HEAD is a text file and each file path is on a new line. If the format is "parquet",
the table schema is two-column ("HEAD name", "file path").

## Results

These are the results of running the tool on PGA'19:

- [configs.tar.xz](https://drive.google.com/open?id=1_cij4BMrPiKVBVdZyUzg1iOhB3pL6EPR) - raw git config files for each siva.
- [heads.csv.xz](https://drive.google.com/open?id=136vsGWfIwfd0IrAdfphIU6lkMmme4-Pj) - mapping from HEAD UUID to repository name.

## License

[MIT](https://tldrlegal.com/license/mit-license).
