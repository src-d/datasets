Commit Messages ![size 46GB](https://img.shields.io/badge/size-46GB-green.svg)
===============

[Download link.](https://drive.google.com/open?id=1Os5MKKdpNUsWUN_vrP23PjsLCq-VpIev)

1.3 billion commit messages extracted from [GHTorrent](https://ghtorrent.org) dumps.

The dataset consists of 3 files:

1. `commits.bin` - commit hashes, 24GB.
2. `repos.txt.xz` - GitHub repository names, 5GB.
3. `messages.txt.xz` - commit messages, 17GB.

The precise number of commits is 1288456749. **There can be duplicate commits. Please contribute a deduplicated dataset if you can.**

### Format

1. `commits.bin` - continuous binary stream, 20 bytes per commit hash. The hashes are random by definition, so it makes no sense to compress this file.
2. `repos.txt.xz` - strings separated by `\0` - NULL character, xz-compressed. The order matches `commits.bin`. There is a trailing '\0'.
3. `messages.txt.xz` - strings separated by `\0`, xz-compressed. The order matches `commits.bin`. There is a trailing '\0'.

### Sample code

Python:
```python
import lzma
from custom_newline import CustomNewlineReader

with open("commits.bin", "rb") as commf:
    with CustomNewlineReader(xz.open("repos.txt.xz"), b"\0") as reposf:
        with CustomNewlineReader(xz.open("messages.txt.xz"), b"\0") as msgf:
            for msg, repo in zip(msgf, reposf):
                commit = commf.read(20).hex()
                print(commit, repo.decode(), msg.decode())
                
```

[`custom_newline.py`](custom_newline.py) is included into this repository.

### Origin

GHTorrent MongoDB dumps before 2019-03-18. The command to generate the dataset was:

```
(
  for dd in 2019-03-17 2019-03-16 ... 2015-12-01; do
    wget -O - http://ghtorrent-downloads.ewi.tudelft.nl/mongo-daily/mongo-dump-$dd.tar.gz |
    tar -xzO dump/github/commits.bson
  done
  for dd in 2015-12-01 2015-10-03 2015-08-03; do
    wget -O - http://ghtorrent-downloads.ewi.tudelft.nl/mongo-full/commits-dump.$dd.tar.gz |
    tar -xzO dump/github/commits.bson
  done
  wget -O - http://ghtorrent-downloads.ewi.tudelft.nl/mongo-full/commits-1-dump.2015-08-04.tar.gz |
  tar -xzO dump/github/commits.bson
) | python3 parse.py
```

`2019-03-17 2019-03-16 ... 2015-12-01` are the dump dates from [ghtorrent.org/downloads.html](http://ghtorrent.org/downloads.html).
[`parse.py`](parse.py) is included into this repository.

### License

[Open Data Commons Open Database License (ODbL)](https://opendatacommons.org/licenses/odbl/)
