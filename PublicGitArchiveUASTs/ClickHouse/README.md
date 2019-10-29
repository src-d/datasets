UASTs extracted from Public Git Archive - ClickHouse DB format ![size 151GB](https://img.shields.io/badge/size-120GB-green.svg)
==============================================================

[Download.](https://pga.sourced.tech/uast-clickhouse/latest.tar.xz)

The [Universal Abstract Syntax Trees](https://doc.bblf.sh/uast/uast-specification-v2.html) (UASTs) extracted
from the latest (HEAD) revision of every Git reference contained in [Public GitArchive](../../PublicGitArchive).
The UASTs are flattened (see what it means [here](#schema)) and saved to [ClickHouse](https://clickhouse.yandex/)
- very performant open source DB for analytics.

The dataset is distributed in the format of ClickHouse DB binary tables, version 19.15.3.6, compressed
with xz. There are two tables, `uasts` and `meta`. See the [schema](#schema) section for details.
There are 24,077,892,285 rows.

### How to install

Assuming that you already have ClickHouse installed, you should do the following steps:

1. Stop the ClickHouse server if it is running.
2. Locate the DB storage root. It is the value of `<path>` in `/etc/clickhouse-server/config.xml`.
3. Unpack the contents of the downloaded archive to a temporary directory.
4. Fix the permissions: the unpacked files must belong to the `clickhouse` user. For example, `sudo chown -R clickhouse:clickhouse /path/to/unpacked/root`.
5. Move the contents of `data` under `<path>/data/<db name>`. `<db name>` is the name of the database
in where you want the UASTs to be located. For example, `default`.
6. Move the contents of `metadata` under `<path>/metadata/<db name>`. The `<db name>` must match the one you used in step 5.
7. Start the ClickHouse server and run a sample query, e.g. `SELECT COUNT(*) from uasts;`.

### Schema

The UASTs are "flattened". This means that only the nodes which have a "value" were left, and the
tree structure is saved in the `parents` array. Thus it is impossible to reconstruct the original
UAST, the conversion is (very) lossy. On the other side, it becomes very easy to mine the dataset.
See the [sample queries](#sample-queries) and the [limitations](#limitations).

```
ATTACH TABLE uasts
(
    `id` Int32, 
    `left` Int32, 
    `right` Int32, 
    `repo` String, 
    `lang` String, 
    `file` String, 
    `line` Int32, 
    `parents` Array(Int32), 
    `pkey` String, 
    `roles` Array(Int16), 
    `type` String, 
    `orig_type` String, 
    `uptypes` Array(String), 
    `value` String, 
    INDEX lang lang TYPE set(0) GRANULARITY 1, 
    INDEX type type TYPE set(0) GRANULARITY 1, 
    INDEX value_exact value TYPE bloom_filter() GRANULARITY 1, 
    INDEX left (repo, file, left) TYPE minmax GRANULARITY 1, 
    INDEX right (repo, file, right) TYPE minmax GRANULARITY 1, 
    INDEX orig_type orig_type TYPE set(0) GRANULARITY 1
)
ENGINE = MergeTree()
ORDER BY (repo, file, id)
```

The "meta" table mirrors the PGA index file.

```
ATTACH TABLE meta
(
    `repo` String, 
    `siva_filenames` Array(String), 
    `file_count` Int32, 
    `langs` Array(String), 
    `langs_bytes_count` Array(UInt32), 
    `langs_lines_count` Array(UInt32), 
    `langs_files_count` Array(UInt32), 
    `commits_count` Int32, 
    `branches_count` Int32, 
    `forks_count` Int32, 
    `empty_lines_count` Array(UInt32), 
    `code_lines_count` Array(UInt32), 
    `comment_lines_count` Array(UInt32), 
    `license_names` Array(String), 
    `license_confidences` Array(Float32), 
    `stars` Int32, 
    `size` Int64, 
    INDEX stars stars TYPE minmax GRANULARITY 1
)
ENGINE = MergeTree()
ORDER BY repo
```

### Sample queries

The cool property of ClickHouse is that the following sample queries finish within a minute
on a single machine with 64 vcores and RAID-0 over NVMe SSDs.

```
# Counting the number of distinct files and repositories

SELECT COUNT(DISTINCT repo) AS repo_count, 
       COUNT(DISTINCT repo, file) AS file_count
FROM uasts;

# Extracting all C# keywords: 

SELECT * 
FROM uasts
WHERE lang = 'csharp'
    AND type = 'Keyword';

# Extracting all comments in files from source{d} repositories:

SELECT * 
FROM uasts
WHERE repo LIKE 'src-d/%'
    AND type = 'Comment';

# Extracting all identifiers from Go files, excluding vendoring files

SELECT * 
FROM uasts
WHERE lang = 'go'
    AND file NOT LIKE '%vendor/%'
    AND type = 'Identifier';
```

For more complex queries, e.g. extracting imports, some tricks are required (see [limitations](#limitations)) - however it can be [done](https://github.com/src-d/ml-mining).


### Origin

The DB was generated using [src-d/uast2clickhouse](https://github.com/src-d/uast2clickhouse) tool
from the original [Public GitArchive UASTs in Parquet](..).
The procedure took 3 days on 16 2-vcore, 16GB worker instances and a 64-vcore, 58GB server instance on
Google Cloud.

### Limitations

As was already mentioned in the [schema section](#schema), the UASTs are flattened in a lossy way.
This includes the aggressive normalization of the data for each programming language. While we did our
best at detecting the possible problems early on, a few sneaked inside when it was too late to fix them. Furthermore, the dataset was not updated after the 21st of september, so any commit merged after will not have been included.

Below is an incomplete list of known issues:

* Duplication: 
    * Some repositories were processed multiple times, because they appeared in multiple Parquet files. We created the DB by iterating on each Parquet file, and did not check for duplicate rows.
    * Errors inherited from Babelfish that are language-specific, e.g. [duplicate go comments](https://github.com/bblfsh/go-driver/issues/56).
* Missing data:
    * Due to how we traversed the trees, some useful information was erased, e.g. all Ruby imports were unfortunately discarded. See [here](https://github.com/src-d/uast2clickhouse/issues/11).
    * Errors inherited from Babelfish that are language-specific, e.g. some drivers do not keep keywords.
* Wrong ordering: the `left` and the `right` columns are sometimes unreliable, due to how we traversed the UASTs and propagated positional information. The line numbers should always be correct though.

If you notice something strange or have trouble using the dataset, please speak up [here](https://github.com/src-d/uast2clickhouse/issues) and we will help to find a workaround. The dump will not be updated until Public Git Archive v3 is released in 2020.

### License

Tools: [Apache 2.0](https://choosealicense.com/licenses/apache-2.0/).
Compilation: [Open Data Commons Open Database License (ODbL)](https://opendatacommons.org/licenses/odbl/).
Underlying code: © their authors and subject of their licenses.
