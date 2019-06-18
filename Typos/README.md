Typos in Identifiers ![size 1MB](https://img.shields.io/badge/size-1MB-green.svg)
====================

[Download.](typos.csv)

[Download the raw YAML candidates.](candidates.tar.xz)

[Mirror on data.world](https://data.world/source-d/typos)

7375 typos developers made in source code identifiers, e.g. class names, function names, variable names,
and fixed them on GitHub. See the [Origin](#Origin) section about how they were mined.

The dataset consists of 2 files:

1. [`typos.csv`](typos.csv) - CSV with the typos.
2. [`candidates.tar.xz`](candidates.tar.xz) - typo candidates, YAML file per suspicious commit.

### Format

Typos CSV has the following columns:

1. `repository` - GitHub repository name.
2. `wrong` - typoed identifier.
3. `correct` - fixed identifier.
4. `commit` - fixing commit hash.
5. `file` - file name where the typo was fixed.
6. `line` - line number in the file where the typo was fixed.

Please note that the information about **where** the typo was introduced is not included.

The candidate YAML format is self-descriptive. It is the raw output from [`hercules --typos-dataset`](https://github.com/src-d/hercules).

### Use cases

- Improve your IDE typos correction by leveraging what other people usually fix.
- We are using this dataset to evaluate [src-d/style-analyzer](https://github.com/src-d/style-analyzer)'s typos correction for assited code reviews on GitHub.
- Any crazy research is possible!

### Origin

The process of mining the typos consists of three stages.

#### Stage 1 - regexp over commit messages

We take the [1.3 billion commit messages dataset](../CommitMessages) and leave only those commits which have
messages satisfying a certain regular expression. Here it is, case-insensitive mode:

```
((fix|correct)(|ed)\s+(|a\s+|the\s+)(typo|misprint)s?\s+.*(func|function|method|var|variable|cls|class|struct|identifier|attr|attribute|prop|property|name))|(^s/[^/]+/[^/]+)
```

We further remove commits by the blacklist regular expression

```
filename|file name|\spath|\scomment
```

The goal of this stage is to leave the commits which have a high chance of fixing a typo in an identifier name.
According to our calculations, this is about 23 times better than picking commits at random.
The number of perspective commits appears to be 192664.
The scripts should be run one after another: [`stage1_1.py`](stage1_1.py) and [`stage1_2.py`](stage1_2.py).

#### Stage 2 - run typo candidates extraction

We run `hercules --typos-dataset` over each commit left after stage 1. This is the most time-consuming task,
it took one month to finish on 4 machines with 1gbps internet, 12 cores and 32 GB RAM.
[Hercules](https://github.com/src-d/hercules) uses [Babelfish](https://docs.sourced.tech/babelfish)
to parse the supported languages, and we had to blacklist certain repositories due to exceeding the available memory.
Particularly, we excluded repos ending with

```
/chromium
/freebsd
/llvm
/linux
/iontrail
/gecko
/main-silver
/openbsd-src
/opal-voip
/gcc
/UnrealEngine
```
The script is included as [`stage2.ipython`](stage2.ipython).
`--typos-dataset` looks at sequentially removed and inserted lines with the same logical identifier,
measures the Levenshtein distance between each pair, and if it is less than a certain threshold (4),
considers it to be a typo.

We obtain 10137 YAML files. We call them the "candidates" and they are included into this dataset.

#### Stage 3 - remove false positives

We run another script to remove irrelevant identifier renames, e.g. "one" -> "two", which were
detected by Hercules due to the way its extraction algorithm works.
The code is [`stage3.py`](stage3.py).
We share the original YAML candidates in case you want to postprocess differently.

### Limitations

We did not review the dataset manually, so there can be irrelevant identifier renames even after applying our multi-stage filtering.
If you spot one, please create a PR with the CSV fix.

### License

Code: [MIT](https://choosealicense.com/licenses/mit/).
Compilation: [Open Data Commons Open Database License (ODbL)](https://opendatacommons.org/licenses/odbl/).
Actual typos: © their authors. If you are an author of a typo and you are ashamed of it, feel free to request its removal in the issues.
