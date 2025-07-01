# Opsicle

Modern operations runbook automation tool.


---



## Deployment

### VM non-container-based deployment

Run the following on any VM with `golang` installed:

```sh
make binary;
./bin/opsicle start standalone \
  --config ./tests/config/basic;
```

### VM container-based deployment

Run the following on any VM with Docker Compose installed:

```sh
docker compose up;
```

### Kubernetes deployment

For local deployments, run the following first:

```sh
make local-k8s-cluster;
```

Run the following to deploy once you've verified the Kubernetes context is correct:

```sh
helm upgrade \
  --install \
  --create-namespace \
  --namespace opsicle \
  --values ./charts/opsicle/values.yaml \
  ./charts/opsicle \
  opsicle;
```


---



## Local development

### Start

```sh
opsicle start standalone
```

### Start with configuration override

```sh
opsicle start standalone \
  --config-dir ./path/to/manifests;
```

### Submit Template

```sh
opsicle create template --source ./path/to/automation/template
```

### Execute Automation

```sh
opsicle run template ${TEMPLATE_NAME}
```

### List Available Automations

```sh
opsicle get templates;
```


---



## Main Success Scenaros

### Platform Engineer can deploy on Kubernetes using Helm

Specification type: Non-functional
Specification priority: 2

1. Platform Engineer clones repository to the VM
2. Platform Engineer runs `helm upgrade --install --namespace opsicle --values ./charts/opsicle/values.yaml ./charts/opsicle opsicle` from the root of the repository
3. Platform Engineer should be able to observe all required components running in the Kubernetes cluster

### Platform Engineer can deploy on a VM using Docker Compose

Specification type: Non-functional
Specification priority: 2

1. Platform Engineer clones repository to the VM
2. Platform Engineer runs `docker compose up -d` from the root of the repository
3. Platform Engineer should be able to observe all required components running

### Platform Engineer can deploy on a VM using just the binary

Specification type: Non-functional
Specification priority: 1

1. Platform Engineer clones repository to the VM
2. Platform Engineer runs `make binary`
3. Platform Engineer can run the following to setup a fully functional deployment:
   ```sh
   chmod +x ./bin/opsicle;
   ./bin/opsicle start standalone;
   ```

### Dev Team Operator is able to trigger a runbook automation

Specification type: Functional
Specification priority: 1

1. Operator logs into Opsicle
2. Operator searches for a runbook automation from the search bar
3. Operator clicks on runbook automation to trigger
4. Operator fills up variables/parameters (if any)
5. Operator verifies their identity using MFA
6. Runbook automation is triggered
7. Operator receives notifications that automation is running
7. Operator receives notifications that automation has completed

### Dev Team Developer is able to trigger a runbook automation

Specification type: Functional
Specification priority: 1

1. Developer logs into Opsicle
2. Developer searches for a runbook automation from the search bar
3. Developer clicks on runbook automation to trigger
4. Developer fills up variables/parameters (if any)
5. Developer verifies their identity using MFA
6. Approver receives notification requesting for approval to run automation
7. Approver approves the approval request
8. Runbook automation is triggered
9. Manifest-defined channel receives notifications that automation is running
10. Manifest-defined channel receives notifications that automation has completed

### Dev Team User is able to debug runbook automation

Specification type: Functional
Specification priority: 1

1. User receives a notification that automation has failed
2. User logs into Opsicle
3. User clicks on a "Runs" menu item
4. User identifies the runbook automation they triggered and clicks into it
5. User sees logs from the runbook automation


---



## Resource Types

1. All resource definitions are written in YAML
2. Resource definitions can be stored locally in the filesystem and loaded on start via flags for a static system

### Runbook Automation Templates

These define runbook automations and can be submitted via the Kubernetes CRD for Kubernetes deployments, or via API for non-Kubernetes-based deployments. Templates will be stored and versioned in the filesystem if no database is defined, otherwise they will be stored in the defined database.

A sample runbook automation template is as follows:

```yaml
apiVersion: v1
type: AutomationTemplate
metadata:
  name: basic
  labels:
    opsicle/description: "this is a basic automation template resource"
spec:
  metadata:
    displayName: Basic Automation
    owners:
    - name: Bjarne
      email: stroustrup@opsicle.io
    - name: Dennis
      email: ritchie@opsicle.io
  template:
    volumeMounts:
    - host: ./tmp
      container: /tmp
    phases:
    - name: initialisation
      image: alpine:latest
      commands:
        - echo "initialisation"
    - name: information-gathering
      image: alpine:latest
      commands:
        - nslookup google.com
    - name: execution
      image: alpine:latest
      commands:
        - mkdir ./tmp
        - wget -qO - google.com > /tmp/google.com.txt
    - name: clean-up
      image: alpine:latest
      commands:
        - cat /tmp/google.com.txt
        - echo "done"
```

### Automation Manifest

```yaml
apiVersion: v1
type: Automation
metadata:
  name: basic
spec:
  status:
    startedAt: 2025-06-28 15:17:39 +0800
  phases:
  - name: initialisation
    image: alpine:latest
    commands:
      - echo "initialisation"
  - name: information-gathering
    image: alpine:latest
    commands:
      - nslookup httpbin.org
  - name: execution
    image: alpine:latest
    commands:
      - wget -qO - https://httpbin.org/ip > /tmp/httpbin.org.ip.txt
  - name: clean-up
    image: alpine:latest
    commands:
      - cat /tmp/httpbin.org.ip.txt
      - echo "done"
  volumeMounts:
  - host: ./tmp
    container: /tmp
```

### Access Templates

```yaml
apiVersion: v1
type: User
metadata:
  name: user-one
spec:
  additionalMetadata:
    displayName: User One
    description: |
      This is user one
  email: babbage@opsicle.com
```

```yaml
apiVersion: v1
type: Group
metadata:
  name: group-one
spec:
  additionalMetadata:
    displayName: Group One
    description: |
      This is group one
```

```yaml
apiVersion: v1
type: GroupAssignment
metadata:
  name: group-assignment-one
spec:
  additonalMetadata:
    name: Group Assignment One
    description: |
      Assigns users to Group One
  group: group-one
  users:
  - user-one
```

### Notification Channels

Notification Channels defne channels where communication happens. This could be approvals or it could be alerts

The following shows a channel using Slack bot:

```yaml
apiVersion: v1
type: NotificationChannel
metadata:
  name: slack
spec:
  type: slack
  slack:
    botToken: %__SLACK_BOT_TOKEN__%
    channelId: %__SLACK_CHANNEL_ID__%
```

The following shows a channel using Slack webhook:

```yaml
apiVersion: v1
type: NotificationChannel
metadata:
  name: slack-webhook
spec:
  type: slack-webhook
  slack-webhook:
    url: %__SLACK_WEBHOOK_URL__%
```

The following shows a channel using Telegram

```yaml
apiVersion: v1
type: NotificationChannel
metadata:
  name: telegram
spec:
  type: telegram
  telegram:
    botToken: %__TELEGRAM_BOT_TOKEN__%
    chatId: %__TELEGRAM_CHANNEL_ID__%
```

### Permissions Manifest

Permissiosn Manifests enable linking permissions to groups or users

```yaml
apiVersion: v1
type: PermissionSet
metadata:
  name: permset-one
spec:
  additionalMetadata:
    displayName: Permission Set 1
    description: |
      Initial permission set
  groups:
  - group-one
  rules:
  - effect: allow
    verbs:
    - execute
    resource:
      tags:
        team: backend
  - effect: approval
    verbs:
    - execute
    resource:
      tags:
        team: frontend
```

### Secret Manifest

Secret Manifests enable operators to define secrets externally from the application and provides a way to separate configuration from secrets management.

```yaml
apiVersion: v1
type: SecretSet
metadata:
  name: app-one-secrets
spec:
  additionalMetadata:
    displayName: AppOne Secrets
    description: |
      Secrets for use with AppOne-based runbook automations
  secrets:
  - type: aws-secretsmanager
    source: arn:aws:secretsmanager:us-east-1:123456789012:secret:app-one-secrets
    use:
    - key: SECRET_KEY_ONE
      as: SECRET_KEY_1
    - key: SECRET_KEY_TWO
      as: SECRET_KEY_2
  - type: filesystem
    source: /path/to/secret
    key: FILESYSTEM_SECRET
  - type: environment
    source: SECRET_KEY_FROM_ENV
    key: ENVVAR_SECRET

```
