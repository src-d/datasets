# GitHub Pull Request Review Comments ![size 1.6GB](https://img.shields.io/badge/size-1.6GB-green.svg)===========

The dataset was extracted from [GH Archive](https://www.gharchive.org/) and consists of:

1. [25.3 million pull request review comments](https://drive.google.com/open?id=1rk6OTDrD09xVU0o_w8_dvtsLaeeUgwmP) since January 2015 till December 2018 - 1.6 GB (xz-compressed)

### Format

CSV, columns:

* `COMMENT_ID` - identifier of the comment in mother dataset - [GH Archive](https://www.gharchive.org/)
* `COMMIT_ID` - commit hash to which the review comment is attached
* `URL` - path to the GitHub pull request the comment comes from
* `AUTHOR` - GitHub user of the author of the comment
* `CREATED_AT` - creation date of the comment
* `BODY` - raw content of the comment

### License

[Open Data Commons Open Database License (ODbL)](https://opendatacommons.org/licenses/odbl/)
