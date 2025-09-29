# Opsicle

Opsicle is a Runbhook Automation platform.

- [Opsicle](#opsicle)
- [Documentation](#documentation)
  - [For Contributors](#for-contributors)
    - [Quicklinks](#quicklinks)
    - [Setting up locally](#setting-up-locally)
      - [Secrets for local services](#secrets-for-local-services)
      - [System architecture overview](#system-architecture-overview)
        - [Notes on system components and how they interact](#notes-on-system-components-and-how-they-interact)
      - [3rd party services overview/setup/notes](#3rd-party-services-overviewsetupnotes)
        - [MongoDB](#mongodb)
        - [MySQL](#mysql)
        - [NATS](#nats)
        - [Redis](#redis)
      - [Scripts and how-tos](#scripts-and-how-tos)
        - [Resetting the database](#resetting-the-database)
  - [For Users](#for-users)
    - [Deploying Opsicle](#deploying-opsicle)
    - [Initialising Opsicle](#initialising-opsicle)

# Documentation

## For Contributors

### Quicklinks

- [./docs/changelog/README.md](Changelog)
- [./docs/integrations.md](Integrations)
- [./docs/ideas.md](Idea log)
- [./docs/system-architecture.md](System Architecture)
- [./docs/testing.md](Testing)

### Setting up locally

1. Make a clone of `.envrc.sample` into `.envrc`
2. Fill up the values according to the values in the password manager

#### Secrets for local services

1. Majority of secrets for local services are stored in the Git repository for ease of setup for local development
2. Where applicable, login credentials should be set to `opsicle:password`
   1. Keep all usernames as `opsicle` as far as possible
   2. Where validations apply and cannot be gotten around, development passwords should be `p@ssw0rd!!Opsicle`
3. The risk of using these credentials is acknowledged and irrelevant to the security of deployed environments since we deploy differently from how services are run locally

#### System architecture overview

Opsicle consists of four main components:

1. CLI tool (`opsicle`) with the code root found at `./cmd/opsicle`
2. Controller (`controller`) which is started via the CLI tool with the code root found at `./cmd/opsicle/start/controller`
2. Coordinator (`coordinator`) which is started via the CLI tool with the code root found at `./cmd/opsicle/start/coordinator`
2. Worker (`worker`) which is started via the CLI tool with the code root found at `./cmd/opsicle/start/worker`

##### Notes on system components and how they interact

1. The CLI tool is the user interface for the Opsicle system
2. The `controller` handles requests from the CLI tool and does most of the platform-level operations/transactions. It's a RESTful API and its SDK can be found at `./pkg/controller`
3. The `coordinator` is a separate service for `worker` service instances to connect to to pull automations to execute. This service is also responsible for communicating all affairs with the `worker` back to the `controller` in an event-driven way via an appointed queue system (NATS).
4. The `worker` component executes automations and reports on their status to the `coordinator` service via gRPC streams.

#### 3rd party services overview/setup/notes

##### MongoDB

MongoDB is used as the audit database and used to store audit logs for all user and system actions.

1. [MongoDB Compass](https://www.mongodb.com/products/tools/compass) is the recommended tool for working with MongoDB. Download it from [this link](https://www.mongodb.com/try/download/compass)
2. To update the username/password locally, you need to remove the entire data directory and start MongoDB from scratch

##### MySQL

MySQL is used as the platform database and used to persist system data.

1. [MySQL Workbench](https://www.mysql.com/products/workbench/) is the recommended tool for working with MySQL. Download it from [this link](https://dev.mysql.com/downloads/workbench/)

##### NATS

NATS is used as the queue system for submitting automations and processing them.

1. [NATS UI](https://natsnui.app/) (included in the `docker-compose` setup) is the recommended tool for working with NATS. After spinning up Docker Compose, you can access it at [http://localhost:31311](http://localhost:31311)

##### Redis

Redis is used as a cache for all internal systems (ie. `controller` and `coordinator`)

1. [Redis Insight](https://redis.io/insight/) (included in the `docker-compose` setup) is the recommended tool for working with Redis. After spinning up Docker Compose, you can access it at [http://localhost:5540](http://localhost:5540)
2. To generate the password seen in the `redis.conf` file, use `printf -- '${THE_PASSWORD_YOU_WANT} | sha256sum -'

#### Scripts and how-tos

Scripts for use on your local machine are avaialble via a `Makefile`.

##### Resetting the database

```sh
make mysql_reset;
```

## For Users

The following instructions assume a deployment where the deployment is accessible over `localhost` or `127.0.0.1`. You may need to modify the URLs to hit the correct server on the correct network relative to your workstation.

### Deploying Opsicle

### Initialising Opsicle

1. Verify that the approver serivce is running at `http://localhost:12345`
   ```sh
   nc -zv 127.0.0.1 12345;
   ```
2. Verify that the controller serivce is running at `http://localhost:54321`
   ```sh
   nc -zv 127.0.0.1 54321;
   ```
3. Verify the database is running
   1. Verify that a MySQL database is available at `127.0.0.1:3306`
      ```sh
      nc -zv 127.0.0.1 3306;
      ```
   2. Verify that you can get a shell via `mysql`
      ```sh
      mysql -uopsicle -h127.0.0.1 -P3306 -ppassword opsicle;
      ```
4. Verify that a cache is running
   1. Verify that a Redis cache is available at `127.0.0.1:6379`
      ```sh
      redis-cli -h 127.0.0.1 -p 6379 --user opsicle --pass password -n 0;
      ```
   2. Verify that you can get a shell via 
5. Create the `root` organisation and superuser
   ```sh
   opsicle init controller;
   ```
6. Login as the user:
   ```sh
   opsicle login;
   ```
