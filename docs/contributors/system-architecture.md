# System Architecture

This page documents the architecture of Opsicle at various levels

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
