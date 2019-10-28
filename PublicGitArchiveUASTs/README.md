UASTs extracted from Public Git Archive ![size 5TB](https://img.shields.io/badge/size-5TB-green.svg)
=======================================

The [Universal Abstract Syntax Trees](https://doc.bblf.sh/uast/uast-specification-v2.html) (UASTs) extracted
from the latest (HEAD) revision of every Git reference contained in [Public GitArchive](../../PublicGitArchive).
The dataset is distributed as Parquet files, which you can download using the [pga CLI](../PublicGitArchive/pga).
There is also a [ClickHouse DB version](ClickHouse) which is more lightweight and easier to work with.

### Format

TODO for Romain: describe only the format of the dataset here. Which columns? Types?

The Parquet files were created by using the [pga2uast](../PublicGitArchive/pga2uast) on the HEAD commit of each repository in the original dataset. The tables below should provide insights on the contents of the dataset.

### Usage

Each row in the parquet files contains the UAST of one file, alongside the filepath and the UUID of
the repository. You can use [this mapping](https://drive.google.com/open?id=136vsGWfIwfd0IrAdfphIU6lkMmme4-Pj)
to obtain the repository names from UUIDs.

The Parquet files can be read using any library that supports the format, however in the case where
you want to process large amounts of data we recommend using Spark. The UASTs are stored as byte arrays,
and thus you can use any of the [Babelfish Clients](https://doc.bblf.sh/using-babelfish/clients.html)
to read and manipulate them.

TODO for Romain: actually write a code snippet to read a Parquet file and show smth from it.

### Origin

We used [pga2uast](../PublicGitArchive/pga2uast) to parse the files in HEAD revisions of PGA repositories.
Please refer to [this GitHub issue](https://github.com/src-d/ml-backlog/issues/74) that describes
the procedure in high detail. It was quite sophisticated because we wanted to cover as much data as we could.
We used 11 "Start-2-L" machines on online.net.

### Limitations - TODO for Romain

|             | # of repos | # of files | # of distinct files | % of duplicates |
|:-----------:|:------------:|:----------:|:--------------------:|:-----------------:|
| **PGA** |  220,174    | 40,971,787 |          40,829,244 | 0.3 %  |
| **UASTs**   |  218,023    | 36,162,330 |          35,991,340 | 0.5 %  |

As you see, we were not able to process 100% of the HEAD of Public Git Archive. For one, we could not process all languages, as Babelfish currently only has drivers for 9 languages - and like all software, it not immune to errors. Furthermore, some repositories proved too large to be processable in a reasonnable amount of time.

|                | file count | file extraction % | file size | byte extraction % |
|:--------------:|:----------:|:--------------------:|:-----------------:|:--:|
|     **ALL**     |  35,991,340 |        88.15 %      |       484.7 GB |        65.37 %       |
||
|     **Go**     |  4,126,578 |        99.88 %       |       56.48 GB |        96.12 %       |
|   **Python**   |  2,994,169 |        89.70 %       |      22.84 GB |        84.36 %       | 
|     **C++**    |  8,726,368 |        80.41 %       |      92.85 GB |        63.69 %       |
|     **C#**     |  2,379,754 |        98.99 %       |      15.43 GB |        93.12 %       | 
|    **Java**    |  6,985,742 |        96.85 %       |     42.19 GB |        95.26 %       | 
| **JavaScript** | 10,466,131 |        80.54 %       | 227.68 GB |        50.09 %       |
|    **Ruby**    |  1,143,654 |        96.70 %       |  3.42 GB  |        91.56 %       |
|     **PHP**    |  2,888,395 |        87.64 %       |     15.55 GB |        71.92 %       | 
|    **Shell**   |  1,118,453 |        87.54 %       |   8.26 GB  |        25.97 %       |

### License

Tools: [Apache 2.0](https://choosealicense.com/licenses/apache-2.0/).
Compilation: [Open Data Commons Open Database License (ODbL)](https://opendatacommons.org/licenses/odbl/).
Underlying code: © their authors and subject of their licenses.