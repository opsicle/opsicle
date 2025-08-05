# Documentation

## Overview

1. Documentation is done using [Docsify.js](https://docsify.js.org)
2. Documentation can be found in the Github repository at the `./docs` path

## Local development

To spin up the documentation locally, first install Docsify.js:

```sh
npm install docsify-cli --global;
```

Then serve the `./docs` directory by running:

```sh
docsify serve ./docs;
```

## CLI documentation generation

The CLI documentation is done via the `github.com/spf13/cobra/docs` package. To generate the documentation, run:

```sh
go run ./cmd/docsgen;
```
