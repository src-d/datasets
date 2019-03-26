# GitHub Pull Request Review Comments ![size 1.5GB](https://img.shields.io/badge/size-1.5GB-green.svg)

[Download link.](https://drive.google.com/file/d/1t-3ZwprNqBHZhO2_Mhp0Y1bCX9M500SM)

25.3 million pull request review comments on GitHub since January 2015 till December 2018.

### Format

xz-compressed CSV, with columns:

* `COMMENT_ID` - identifier of the comment in mother dataset - [GH Archive](https://www.gharchive.org/)
* `COMMIT_ID` - commit hash to which the review comment is attached
* `URL` - path to the GitHub pull request the comment comes from
* `AUTHOR` - GitHub user of the author of the comment
* `CREATED_AT` - creation date of the comment
* `BODY` - raw content of the comment

### Sample code

Python:
```python
# too big for pandas.read_csv
import codecs
import csv
import lzma

with lzma.open("review_comments.csv.xz") as archf:
    reader = csv.DictReader(codecs.getreader("utf-8")(archf))
    for record in reader:
        print(record)
```

### Origin

The dataset was generated from [GH Archive](https://www.gharchive.org/) in the [following notebook](PR_review_comments_generation.ipynb).
The comments which exceeded Python's `csv.field_size_limit` equal to 128KB were discarded (~10 comments).

We gathered some [statistics about the dataset](PR_review_comments_stats.ipynb).

### License

[Open Data Commons Open Database License (ODbL)](https://opendatacommons.org/licenses/odbl/)
