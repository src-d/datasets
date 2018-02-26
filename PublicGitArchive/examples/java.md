Downloading all Java repositories
=================================

Execute the following to download all Java siva files to the current directory:

```bash
./multitool get-index | tee index.csv | grep -P '[",]Java[",]' | grep -oE '[0-9a-f]{40}\.siva' | ./multitool get-dataset -o .
```

`index.csv` contains metadata of all the repositories, `grep -P '[",]Java[",]'` it to list only
Java ones.

Grab `multitool` from the [Releases page](https://github.com/src-d/datasets/releases).
