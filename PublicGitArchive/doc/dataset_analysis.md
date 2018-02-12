# Description of the current dataset

## 1. GHTorrent: selecting the list of URLs

From GHTorrent’s MySQL dump, dated from 01/01/2018.
By querying repos with at least 50 stargazers

Output: list of 190,051 URLs with duplicates  
List of unique URLs: 187,352

## 2. Collection with Borges

Output of Borges:  

183,015 **repositories**  
4,338 are **indefinitely pending** for several reasons :  
* private repos  
* not existing anymore  
* borges is not able to process them, maybe a bug

## 3. Dataset stats in HDFS

Total **size** : 2.7 TB  
Number of **siva files** : 278,923


## 4. Latest CSV file : `dataset_stats.ipynb`

bug?: row nº 171531 contains only the titles of the columns, see `dataset_stats.ipynb`. It is dropped to compute the rest of the stats.

Number of **rows/urls** : 181,476  
Number of **siva files** : 239,800  
Number of **forks** : 56,300

Number of distinct **languages** : 455  
In Linguist/enry, we count 340 languages of type: programming and 469 overall. Thus, it includes also non programming languages.

Total number of **files** : 50.3 million  
Total number of **lines** : 14,751 million  
Total number of **bytes** : 652 GB  

## 5. Dataset Analysis given the latest CSV file : `size_analysis_siva_files.ipynb`

* Repositories that count the most siva files, [related issue](https://github.com/src-d/borges/issues/222)

|   | Number of siva files | URL | Total size | min/max size | Average size |  
|---|----------------------|-----|------------|--------------|--------------|
| 1. | 5222 | github.com/android/platform_bionic |  ~     |  ~  |  ~  |
| 2. | 4623 | github.com/android/platform_build  | 107 MB | 2 KB / 76 MB | 23 KB |  
| 6. | 2467 | github.com/upspin/upspin           | 75 MB  | 2 KB / 59 MB | 30 KB |  

**Conclusion** : Those repos have thousands of references with root commits different than the one in master. However, because they remain tiny, they don't pollute the dataset and we don't filter them.

* And the largest siva files belong to ...  

See `size_analysis_siva_files.ipynb` for the list of the 10 largest siva files in the dataset.

|   | Size of siva files | Repos |
|---|----------------------|-----|
| 1. | 11.2 GB | <ul><li>github.com/spotify/linux</li><li>github.com/linux-pmfs/pmfs</li><li>github.com/ARM-software/linux</li><li>github.com/libos-nuse/net-next-nuse</li></ul> |
| 2. | 6.1 GB | github.com/catpanda/AVcollection  |
| 3. | 5.4 GB | ~          |
| 4. | 5.1 GB | ~          | 
| 5. | 4.9 GB | <ul><li>github.com/arduino-org/Arduino</li><li>github.com/arduino/Arduino</li><li>github.com/adafruit/ESP8266-Arduino</li></ul>   | 
| 6. | 4.9 GB | github.com/sixteencolors/sixteencolors...  | 
| 7. | 4.6 GB | github.com/MicrosoftDocs/azure-docs        | 
| 8. | 4.3 GB | ~          | 
| 9. | 4.1 GB | <ul><li>github.com/adobe/chromium</li><li>github.com/mirrors/chromium</li></ul> | 
| 10.| 3.9 GB | github.com/dotabuff/d2vpk  | 

Some of the links returns 404, others are photo storage... This could be filtered by querying GHTorrent not only with high number of stars but also with the number of contributors, open issues or PRs.
