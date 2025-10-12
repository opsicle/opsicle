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
make docs_site;
```

## CLI documentation generation

The CLI documentation is done via the `github.com/spf13/cobra/docs` package. To generate the documentation, run:

```sh
make docs_cli
```

## Sitemap generation

To generate the sitemap, visit https://docsify-sitemap.js.org/ and use:

| Property | Value |
| --- | --- |
| Website URL | `https://docs.opsicle.io` |
| Repository Owner (Username) | `opsicle` |
| Repository Name | `opsicle` |
| Base Directory | `/docs` |
| Branch | `main` |

Copy and paste the generated `sitemap.xml` into `./docs/sitemap.xml`
