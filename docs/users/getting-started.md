# Getting Started

Hello 👋🏼 and thank you for trying out Opsicle

# Getting Opsicle

You can download our all-in-one Opsicle CLI tool from our Github repository's release page at this link: https://github.com/opsicle/opsicle/releases

# Testing out Opsicle

Copy and paste the following runbook automation into a file at `./automation.yaml`:

```sh
apiVersion: v1
type: AutomationTemplate
metadata:
  name: basic
  labels:
    opsicle.io/description: "this is a basic automation template resource"
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
        - mkdir -p /tmp
        - wget -qO - google.com > /tmp/google.com.txt
    - name: clean-up
      image: alpine:latest
      commands:
        - cat /tmp/google.com.txt
        - echo "cleanup done"
```

You can now run the following to trigger this automation:

```sh
opsicle run ./automation.yaml;
```

# Testing out a fuller Opsicle

## Creating an account

We create an account using the following command (you will be prompted to enter a username and password)

```sh
opsicle register;
```

Check your email, copy the verficiation code and enter it into the prompt that appears after you run:

```sh
opsicle verify email;
```

Login using the email and password you set earlier:

```sh
opsicle login;
```

When you're done you can also logout using:

```sh
opsicle logout;
```

## Creating an organisation

Creating an organisation requires two things:

1. An organisation name (eg. `"Opsicle Inc"`)
2. An organisation codeword (eg. `opsicle`)

The organiastion name will be used for display purposes while the organiastion codeword will be used to identify resources you own.  
- In the cloud version of Opsicle, the organisation codeword will also enable access to your organisation at `codeword.opsicle.cloud`
- In a self-hosted version of Opsicle, the organiastion codeword will enable access to your organisation at `codeword.yourdomain.com` if the deployment allows for it

Run the following command to create an organisation (you will be prompted for the organisation name and codeword):

```sh
opsicle create org;
```

## Creating workers for your organisation

Create an organisation token to use in your worker:

```sh
opsicle create org token;
```

You will receive:

1. API key ID: Identifies your API key and also serves as a multi-factor authentication mechanism
2. API key: Your actual API key

It is recommended to treat both as secrets if you can, but for easier reference, the API key ID can be treated as a non-secret configuration value to help with token identification during token rotation exercises.

Pass this to your self-deployed worker which you can start using:

```sh
opsicle start worker \
   --key-id ${API_KEY_ID} \
   --key ${API_KEY};
```

## Triggering a runbook

Trigger a local runbook automation by running:

```sh
opsicle run automation \
   --org ${ORG_CODE} \
   ./path/to/template.yaml;
```

## Triggering a pre-configured runbook

Create a runbook template for others to run by running:

```sh
opsicle create template \
   --org {ORG_CODE} \
   ./path/to/template.yaml;
```

You can then trigger that runbook automation using:

```sh
opsicle run --org ${ORG_CODE} ${RUNBOOK_NAME};
```

An automation ID will be returned to you which you can use to check the status and retrieve the logs of that automation.

Retrieve the status of the runbook automation execution using:

```sh
opsicle get status ${AUTOMATION_ID};
```

Retrieve the logs of the runbook automation execution:

```sh
opsicle get logs ${AUTOMATION_ID};
```
