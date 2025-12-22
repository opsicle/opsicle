# System Architecture

This page documents the architecture of Opsicle at various levels

Opsicle consists of four main components:

1. CLI tool (`opsicle`) with the code root found at `./cmd/opsicle`
2. Controller (`controller`) which is started via the CLI tool with the code root found at `./cmd/opsicle/start/controller`
2. Coordinator (`coordinator`) which is started via the CLI tool with the code root found at `./cmd/opsicle/start/coordinator`
2. Worker (`worker`) which is started via the CLI tool with the code root found at `./cmd/opsicle/start/worker`

## Components

### CLI tool

- Primary user interface for the Opsicle system

### Controller

- Provides an API for the management console
- 

### Coordinator

- Provides an API for workers to connect to

### Worker

- Runs in a client's production environment

### Data stores

#### MongoDB

MongoDB is used as the audit database and used to store audit logs for all user and system actions.

1. [MongoDB Compass](https://www.mongodb.com/products/tools/compass) is the recommended tool for working with MongoDB. Download it from [this link](https://www.mongodb.com/try/download/compass)
2. To update the username/password locally, you need to remove the entire data directory and start MongoDB from scratch

#### MySQL

MySQL is used as the platform database and used to persist system data.

1. [MySQL Workbench](https://www.mysql.com/products/workbench/) is the recommended tool for working with MySQL. Download it from [this link](https://dev.mysql.com/downloads/workbench/)

#### NATS

NATS is used as the queue system for submitting automations and processing them.

1. [NATS UI](https://natsnui.app/) (included in the `docker-compose` setup) is the recommended tool for working with NATS. After spinning up Docker Compose, you can access it at [http://localhost:31311](http://localhost:31311)

#### Redis

Redis is used as a cache for all internal systems (ie. `controller` and `coordinator`)

1. [Redis Insight](https://redis.io/insight/) (included in the `docker-compose` setup) is the recommended tool for working with Redis. After spinning up Docker Compose, you can access it at [http://localhost:5540](http://localhost:5540)
2. To generate the password seen in the `redis.conf` file, use `printf -- '${THE_PASSWORD_YOU_WANT} | sha256sum -'

## Packages

> Last updated 2025-08-16

```mermaid
flowchart TD;
  ui[UI]
  cmd[CLI Tool]
  controller[Controller Service/API]
  pcontroller[./pkg/controller]
  icontroller[./internal/controller]
  models[./internal/controller/models]
  db[database]

  ui -->|via HTTP| controller
  cmd -->|uses as SDK| pcontroller
  pcontroller -->|provides SDK for| controller
  controller -->|handles routing to handlers in| icontroller
  icontroller -->|uses| models
  models -->|queries| db
  db -.->|responds with data| models
```

## Deployment

> Last updated 2025-08-05


```mermaid
flowchart TD;
  approver[Approver]
  cache[Cache]
  controller[Controller]
  rdbms[RDBMS]
  documentDb[Document DB]
  idp[[Identity Provider]]
  slack[Slack]
  telegram[Telegram]
  webapp[Web Application]
  user((User))
  subgraph Data Persistence
  cache
  documentDb
  rdbms
  end
  subgraph Applications
  controller--->|platform data|rdbms
  controller--->|audit data|documentDb
  controller--->|session info|cache
  approver--->|approval requests|cache
  controller--->|sends approval requests|approver
  approver-.->|pingback via webhook|controller
  webapp--->|via API calls|controller
  end
  subgraph External Systems
  idp
  approver--->|via Bot API|telegram
  approver--->|via Slack App|slack
  controller--->idp
  end
  user--->|via Browser|webapp
  user--->|via CLI tool|controller
```

## Request Flows

### Fully managed model

```mermaid
flowchart TD;
  user((User))
  userManager((User's Manager))
  webApp[Opsicle Web Application]
  clictl[Opsicle CLI Tool]
  appServer[Opsicle Application Server]
  worker[Opsicle Worker]
  job[Job]
  target(Target System)
  approver[Approver]
  subgraph User Local Environment
  user
  clictl
  userManager
  end
  subgraph Opsicle Cloud Environment
  user <-->|via Browser| webApp
  user <-->|via CLI| clictl
  webApp <-->|via API calls| appServer
  clictl <-->|via API calls| appServer
  appServer <-.->|if applicable, gets approval| approver
  approver -->|sends approval request| userManager
  userManager -.->|approves/rejects| approver
  worker -->|polls for Automations| appServer
  appServer -.->|receives Automations| worker
  worker -->|spins up| job
  end
  subgraph Client Environment
  job -->|performs Automation-defined action| target
  end
```


### Shared responsibilities model

This is what the components would look like if you were to subscribe to a cloud plan but want your 

```mermaid
flowchart TD;
  A((User))
  B[Opsicle Web Application]
  C[Opsicle CLI Tool]
  D[Opsicle Application Server]
  E[Opsicle Worker]
  F[Job]
  G[Target System]
  subgraph User Local Environment
  A
  C
  end
  subgraph Opsicle Cloud Environment
  A <-->|via Browser| B
  A <-->|via CLI| C
  B <-->|via API calls| D
  C <-->|via API calls| D
  end
  subgraph Client Environment
  E -->|polls for Automations| D
  D -.->|receives Automations| E
  E -->|spins up| F
  F -->|performs Automation-defined action| G
  end
```

### Self-hosted model

This is what the components would look like if you were to host Opiscle entirely yourself.

```mermaid
flowchart TD;
  A((User))
  B[Opsicle Web Application]
  C[Opsicle CLI Tool]
  D[Opsicle Application Server]
  E[Opsicle Worker]
  F[Job]
  G[Target System]
  subgraph User Local Environment
  A
  C
  end
  subgraph Client Environment
  A <-->|via Browser| B
  A <-->|via CLI| C
  B <-->|via API calls| D
  C <-->|via API calls| D
  E -->|polls for Automations| D
  D -.->|receives Automations| E
  E -->|spins up| F
  F -->|performs Automation-defined action| G
  end
```
