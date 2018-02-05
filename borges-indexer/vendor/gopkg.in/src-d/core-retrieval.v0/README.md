# core-retrieval [![Build Status](https://travis-ci.org/src-d/core-retrieval.svg?branch=master)](https://travis-ci.org/src-d/core-retrieval) [![codecov.io](https://codecov.io/gh/src-d/core-retrieval/branch/master/graph/badge.svg?token=RCW9yo5m4E)](https://codecov.io/gh/src-d/core-retrieval)

**core-retrieval** provides the models and services that are used across
different Data Retrieval projects.

It uses a simple
[Dependency Injection Container](https://en.wikipedia.org/wiki/Dependency_injection)
configured with environment variables.

### Generate models and schema

**core-retrieval** uses [kallax](https://github.com/src-d/go-kallax) as an ORM, and thus, it needs some code generation every time the models change. You can regenerate models by doing:

```
make generate-models
```

Or just:

```
go generate ./...
```

On top of that, the SQL schema of the models included in this package is bundled together as Go code using [go-bindata](https://github.com/jteeuwen/go-bindata). To regenerate the bindata of the schema, just run the following command:

```
make schema
```

There are some checks in the CI that prevent a build from suceeding if both the models and the schema are not generated.

## License

Licensed under the terms of the Apache License Version 2.0. See the `LICENSE`
file for the full license text.

