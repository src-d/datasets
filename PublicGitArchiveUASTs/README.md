UASTs extracted from Public Git Archive ![size 5TB](https://img.shields.io/badge/size-5TB-green.svg)
=======================================

The [Universal Abstract Syntax Trees](https://doc.bblf.sh/uast/uast-specification-v2.html) (UASTs) extracted
from the latest (HEAD) revision of every Git reference contained in [Public Git Archive](../../PublicGitArchive).
The dataset is distributed as Parquet files, which you can download using the [pga CLI](../PublicGitArchive/pga).
There is also a [ClickHouse DB version](ClickHouse) which is more lightweight and easier to work with.

### Format

The Parquet files have 3 columns, one row per file:

- `head` (string): the UUID of the repository of the given file. You can use [this mapping](https://drive.google.com/open?id=136vsGWfIwfd0IrAdfphIU6lkMmme4-Pj)
to obtain the repository names from UUIDs.
- `path` (string): the filepath to the given file, in the repository structure.
- `uast` (variable-length byte array): the UAST of the given file.

### Usage

The Parquet files can be read using any library that supports the format, however using Spark is strongly advised if you need to process a large part of the dataset. The UASTs are stored as byte arrays, and thus you can use any of the [Babelfish client libraries](https://doc.bblf.sh/using-babelfish/clients.html) to read and manipulate them.

For example, this is how to extract all identifiers from the UASTs in a given Parquet file:

```Python
import bblfsh

from pyspark import SparkConf
from pyspark.sql import SparkSession
from pyspark.sql.functions import explode, udf
from pyspark.sql.types import ArrayType, StringType

# We create the Spark Config - tune accordingly

conf = SparkConf().setAll([ ... ])

# We create the Spark Session - the master URL may be wrong depending on your cluster

spark = SparkSession.builder \
    .appName("pga-example") \
    .master("spark://spark-spark-master:7077") \
    .config(conf=conf) \
    .getOrCreate()

# We define the function that will extract identifiers from each UAST

def extract_identifiers(uast):
    ctx = bblfsh.decode(uast)  # Decode the Byte Array and create the Context
    identifiers = []
    for node in ctx.filter("//uast:Identifier"):  # Iterate over the identifier nodes
        node = node.load()  # Load the node in memory
        identifiers.append(node["Name"])  # Extract the identifier from the node
    return identifiers

# We create the Spark User Defined Function usaing above function

extract_identifiers_udf = udf(extract_identifiers, ArrayType(StringType()))

# We apply the pipeline, then trigger execution with `show`

df = spark.read.parquet("/path/to/parquet")
df = df.withColumn("identifier", explode(extract_identifiers_udf(df.uast))) \
    .select("head", "path", "identifier")
df.show()
```

Please note that the [Babelfish Python client library](https://github.com/bblfsh/python-client) needs to be present on the Spark workers for this snippet to function, **not only on the driver.**

### Origin

We used [pga2uast](../PublicGitArchive/pga2uast) to parse the files in HEAD revisions of PGA repositories.
Please refer to [this GitHub issue](https://github.com/src-d/ml-backlog/issues/74) that describes
the procedure in high detail. It was quite sophisticated because we wanted to cover as much data as we could.
We used 11 "Start-2-L" machines on online.net.

### Limitations

|           | # of repos | # of files | # of distinct files | % of duplicates |
|:---------:|:----------:|:----------:|:-------------------:|:---------------:|
| **PGA**   | 220,174    | 40,971,787 | 40,829,244          | 0.3 %           |
| **UASTs** | 218,023    | 36,162,330 | 35,991,340          | 0.5 %           |

As the above table shows, we were not able to process 100% of the HEAD of Public Git Archive. We did not process all the languages because Babelfish currently has drivers for only 9 languages. Furthermore, some files proved to be too large to be processed in a reasonable amount of time. Combined with parsing errors and bugs on Babelfish's side, those resulted in missing ~12% of all parsable files in the HEAD of PGA. They amount for ~45% of all the data in bytes. As we can see from the table below, the distribution of the number of errors by language is not uniform: for instance, the C++ driver, which handles all C-like languages (C, C++, Metal, Cuda), performed worse than the others, while the Go driver performed much better.

|                | # of distinct files processed | % of files processed | # of bytes processed | % of bytes processed |
|:--------------:|:-----------------------------:|:--------------------:|:--------------------:|:--------------------:|
| **All parsable** | 35,991,340                  | 88.15 %              | 484.7 GB             | 65.37 %              |
|                                                                                                                     |
| **Go**           | 4,126,578                   | 99.88 %              | 56.48 GB             | 96.12 %              |
| **Python**       | 2,994,169                   | 89.70 %              | 22.84 GB             | 84.36 %              |
| **C++**          | 8,726,368                   | 80.41 %              | 92.85 GB             | 63.69 %              |
| **C#**           | 2,379,754                   | 98.99 %              | 15.43 GB             | 93.12 %              |
| **Java**         | 6,985,742                   | 96.85 %              | 42.19 GB             | 95.26 %              |
| **JavaScript**   | 10,466,131                  | 80.54 %              | 227.68 GB            | 50.09 %              |
| **Ruby**         | 1,143,654                   | 96.70 %              | 3.42 GB              | 91.56 %              |
| **PHP**          | 2,888,395                   | 87.64 %              | 15.55 GB             | 71.92 %              |
| **Shell**        | 1,118,453                   | 87.54 %              | 8.26 GB              | 25.97 %              |

### License

Tools: [Apache 2.0](https://choosealicense.com/licenses/apache-2.0/).
Compilation: [Open Data Commons Open Database License (ODbL)](https://opendatacommons.org/licenses/odbl/).
Underlying code: © their authors and subject of their licenses.