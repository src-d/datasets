Identifiers ![size 1.0GB](https://img.shields.io/badge/size-1.0GB-green.svg)
===========

[Paper](https://arxiv.org/abs/1805.11651) (accepted to [ML4P'18](https://ml4p.org/)).

The dataset was extracted from [Public Git Archive](https://github.com/src-d/datasets/tree/master/PublicGitArchive) and consists of:

1. [49 million distinct identifiers](https://drive.google.com/open?id=1wZR5zF1GL1fVcA1gZuAN_9rSLd5ssqKV) - 1 GB
2. [identifiers per language](https://drive.google.com/open?id=1dJQVEsLqOQxTsnF9ura-EukRMS9Ew2KJ) - 1 GB, same processing as (1) but extracted from specific programming language files: Python, Javacript, C, C++, PHP, Ruby, C#, Java, Shell, Go, Objective-C.

### Format

CSV, columns:

* `num_files` - number of files where the identifier was found
* `num_occ` - number of times the identifier was found overall
* `num_repos` - number of repositories in which the identifier was found
* `token` - the value of the identifier
* `token_split` - the splitted parts using the [sourced-ml heuristics](https://github.com/src-d/ml/blob/0.5.0/sourced/ml/algorithms/token_parser.py#L71)

All the stats correspond to the HEAD revision of each repository in PGA.

### Code examples

* [Jupyter notebook](https://gist.github.com/zurk/58afacd6da9bf6319eb2839ff8645d99) which reads the per-language identifiers (2) and plots the statistics.

### License

[Open Data Commons Open Database License (ODbL)](https://opendatacommons.org/licenses/odbl/)
