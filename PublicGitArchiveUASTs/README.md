Public Git Archive UASTs
==================

This dataset is stored under two formats:

- as Parquet files, which you can download using the [pga CLI](../PublicGitArchive/pga);
- as a Clickhouse DB dump, which you can download [from here](https://pga.sourced.tech/uast-clickhouse/latest.tar.xz).

## Parquet files ![size 5TB](https://img.shields.io/badge/size-5TB-green.svg)

The Parquet files were created by using the [pga2uast](../PublicGitArchive/pga2uast) on the HEAD commit of each repository in the original dataset. The tables below should provide insights on the contents of the dataset.

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


### Usage

Each row in the parquet files contains the UAST of one file, alongside the filepath and the UUID of the repository. You can use [this mapping](https://drive.google.com/open?id=136vsGWfIwfd0IrAdfphIU6lkMmme4-Pj) to get the repository names from UUIDs.

The Parquet files can be read using any library that supports the format, however in the case where you want to process large amounts of data we recommend using Spark. The UASTs are stored as byte arrays, and thus you can use any of the [Babelfish Clients](https://doc.bblf.sh/using-babelfish/clients.html) to read and manipulate them.

## Clickhouse DB dump ![size 400GB](https://img.shields.io/badge/size-400GB-green.svg)