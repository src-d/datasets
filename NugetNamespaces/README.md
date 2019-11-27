NuGet Namespaces ![size 13MB](https://img.shields.io/badge/size-13MB-green.svg)
================

[Download.](dataset.json.xz)

Namespaces defined in .NET assemblies of 227,839 [NuGet packages](https://www.nuget.org/) extracted in late November 2019.
There is information about 681,858 .NET namespaces overall.
The dataset is an archived JSON file, 145MB uncompressed.

### Format

List with JSON objects, each has exactly 4 properties:

1. `name` - NuGet package name. It should correspond to the package URL: https://www.nuget.org/packages/$(name)
2. `description` - package description from `.nuspec`.
3. `tags` - the list of tags from `.nuspec`.
4. `namespaces` - namespaces where types are defined in the DLLs. The dictionary key is a namespace name and the value is the number of defined types summed through all the assemblies of the package.

`namespaces`, `tags` or `description` may be empty.

### Use cases

Build the mapping from namespaces to package names.

### Origin

We fetched `.nupkg` and `.nuspec` files using [emgarten/NuGet.CatalogReader](https://github.com/emgarten/NuGet.CatalogReader). The exact command was:

```
nugetmirror nupkgs https://api.nuget.org/v3/index.json -o /path/to/packages --latest-only --max-threads 16 --ignore-errors
```

Then we processed each `.nupkg` using [`consumer.py`](consumer.py) - beanstalkd-based namespace extractor. It requires [pystalk](https://github.com/EasyPost/pystalk). There were 5 processes launched with

```
python3 consumer.py -x /path/to/Examples -t /tmp/nuget -o r$(index).json
```

`/path/to/Examples` is the path to hacked `Examples` executable from [0xd4d/dnlib](https://github.com/0xd4d/dnlib). See the patched [`Example1.cs`](Example1.cs) and [`Program.cs`](Program.cs). The tasks were ingested into beanstalkd using [beanstool](https://github.com/src-d/beanstool):

```
for pkg in $(ls /path/to/packages); do ./beanstool put -t default -b "/path/to/packages/$pkg"; done
```

The resulting JSON files were joined together and deduplicated by `name`.

### Limitations

We could not process 492 out of 228,331 downloaded packages. According to a brief error analysis, most of the errors were due to corrupted or invalid nupkg (zip) files.

### License

Code: [MIT](https://choosealicense.com/licenses/mit/).
Data: [Open Data Commons Open Database License (ODbL)](https://opendatacommons.org/licenses/odbl/).
