Downloading all Java repositories
=================================

Execute the following to download all Java siva files to the current directory:

```bash
./multitool get-index | tee index.csv | grep -P '[",]Java[",]' | grep -oP '[0-9a-f]{40}\.siva' | ./multitool get-dataset -o .
```