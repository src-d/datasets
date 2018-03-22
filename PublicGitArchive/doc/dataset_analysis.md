# Description of the current dataset

## 1. GHTorrent: selecting the list of URLs

From GHTorrent’s MySQL dump, dated from 01/01/2018.
By querying repos with at least 50 stargazers

Output: list of 187,352 URLs  

## 2. Collection with Borges

Output of Borges:  

182,014 **repositories**  
5,338 are **indefinitely pending** for several reasons :  
* 3,156 became inaccessible by the Git clone time because :
    * they became private
    * the did not exist anymore
* 82 removed for legal reasons
* borges was not able to process the rest  

## 4. Latest CSV file : `dataset_stats.ipynb`

Number of **rows/urls** : 182,014  
Number of **siva files** : 248,043  
Number of **forks** : 59,246  
Total size : 2.96 TB

Number of distinct **languages** : 455  
In Linguist/enry, we count 340 languages of type: programming and 469 overall. Thus, it includes also non programming languages.

Total number of **files** : 54.5 million  
Total number of **lines** : 15,941 million  
Total number of **bytes** : 710 GB  

## 5. Dataset Analysis given the latest CSV file

* Repositories that count the most siva files, [related issue](https://github.com/src-d/borges/issues/222)

|   | Number of siva files | URL | Total size | min/max size | Average size |  
|---|----------------------|-----|------------|--------------|--------------|
| 1. | 6,613 | github.com/google/angle | 98 MB |  2 KB / 38 MB |  15 KB  |
| 2. | 5,222 | github.com/android/platform_bionic | 85 MB | 2 KB / 48 MB | 16 KB |
| 3. | 4,623 | github.com/android/platform_build  | 107 MB | 2 KB / 76 MB | 23 KB |  

**Conclusion** : Those repos have thousands of references with root commits different than the one in master. However, because they remain tiny, they don't pollute the dataset and we don't filter them.

* And the biggest siva files belong to ...  

List of the 10 biggest siva files in the dataset.

|   | Size of siva files | Repos |
|---|----------------------|-----|
| 1. | 11.2 GB | <ul><li>github.com/spotify/linux</li><li>github.com/linux-pmfs/pmfs</li><li>github.com/ARM-software/linux</li><li>github.com/libos-nuse/net-next-nuse</li><li>github.com/ljalves/linux_media</li><li>github.com/faux123/Nexus_5</li><li>github.com/google/capsicum-linux</li><li>github.com/google/ktsan</li><li>github.com/OpenChannelSSD/linux</li><li>github.com/o11s/open80211s</li><li>github.com/altera-opensource/linux-socfpga</li><li>github.com/sultanxda/android_kernel_oneplus_msm8974</li><li>github.com/NextThingCo/CHIP-linux</li><li>github.com/mjg59/linux</li><li>github.com/rockchip-linux/kernel</li></ul> |
| 2. | 6.1 GB | github.com/catpanda/AVcollection  |
| 3. | 5.4 GB | github.com/google/mysql-protobuf  |
| 4. | 5.1 GB | <ul><li>github.com/Baystation12/Baystation12</li><li>github.com/d3athrow/vgstation13</li></ul>  | 
| 5. | 4.9 GB | <ul><li>github.com/arduino-org/Arduino</li><li>github.com/arduino/Arduino</li><li>github.com/adafruit/ESP8266-Arduino</li></ul>   | 
| 6. | 4.9 GB | github.com/sixteencolors/sixteencolors-archive  | 
| 7. | 4.6 GB | <ul><li>github.com/MicrosoftDocs/azure-docs</li><li>github.com/Azure/azure-content</li></ul> | 
| 8. | 4.1 GB | <ul><li>github.com/adobe/chromium</li><li>github.com/mirrors/chromium</li><li>github.com/ChromiumWebApps/chromium</li></ul> | 
| 9.| 3.9 GB | github.com/dotabuff/d2vpk  | 

Some of the URLs are already deleted.
