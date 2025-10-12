# Getting Started

## Usage

### Account creation

We create an account:

```sh
opsicle register;
```

Check your email and enter the verficiation code:

```sh
opsicle verify email;
```

### Authentication

Login using:

```sh
opsicle login;
```

Logout using:

```sh
opsicle logout;
```

### Organisation creation

Create an organisation to store runbook templates in:

```sh
opsicle create org;
```

### Organisation token creation

Create an organisation token to use in your worker:

```sh
opsicle create org token;
```

You will receive:

1. API key ID
2. API key
3. Certificate PEM
4. Private key PEM

- For automation of user actions, you will need the API key ID and API key
- For initialising a worker to run your automations, you will need the Certificate PEM and Private key PEM

### Worker deployment

Start the local worker with the generated certificates:

```sh
opsicle start worker \
   --cert ./path/to/cert
```

### Template creation

Create a template using:

```sh
opsicle create org template;
```

### Automation creation

Create a runbook automation using:

```sh
opsicle create org automation;
```

### Automation status retrieval

Retrieve the status of the runbook automation using:

```sh
opsicle get org automation;
```

### Automation logs retrieval

Retrieve the status of the runbook automation logs using:

```sh
opsicle get org automation logs;
```

## Deployment

### Pre-requisites

The following data stores are required

#### MySQL Database

#### MongoDB Database

#### Redis Cache

#### NATS Queue

### Deploying via CLI tool

> This method targets VM-based deployments

1. Download the CLI tool
2. Run the following to start the `approver` service
   ```sh
   opsicle start approver
   ```
2. Run the following to start the `controller` service
   ```sh
   opsicle start controller
   ```

### Deploying with Helm

> This method targets Kubernetes-based deployments
