# Idea Log

## Automations

- [ ] Create a library of pre-defined jobs
  - [ ] Cluster scaledown for development environments
  - [ ] Run a port scan to discover all open ports across a CIDR block
  - [ ] Run a service discovery to discover all IP addresses that are in use
  - [ ] Get a list of all pod instances
  - [ ] Trigger a Kubernetes DaemonSet roll-out
  - [ ] Trigger a Kubernetes Deployment roll-out
  - [ ] Trigger a Kubernetes StatefulSet roll-out
  - [ ] Trigger a Job resource based on an existing Cronjob resource

## Approver Service

- Callbacks
  - [ ] Add client certificate authentication as an auth method
  - [ ] Add callback handling for sending an email with a generic SMTP provider
  - [ ] Add callback handling for sending an email with AWS SES
  - [ ] Add callback handling for inserting a AWS SQS message
  - [ ] Add callback handling for inserting a Kafka message
  - [ ] Add callback handling for inserting a NATS message

## Controller Service

## CLI

- [ ] Implement `opsicle login` - Logs the user in
- [ ] Implement `opsicle get automations` - Lists all available automations to the logged-in user
- [ ] Implement `opsicle logout` - Logs the logged-in user out
