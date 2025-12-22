# Local environment setup

## Configure the environment

1. Make a clone of `.envrc.sample` into `.envrc`
2. Fill up the values according to the values in the password manager

### On secrets for local services

1. Majority of secrets for local services are stored in the Git repository for ease of setup for local development
2. Where applicable, login credentials should be set to `opsicle:password`
   1. Keep all usernames as `opsicle` as far as possible
   2. Where validations apply and cannot be gotten around, development passwords should be `p@ssw0rd!!Opsicle`
3. The risk of using these credentials is acknowledged and irrelevant to the security of deployed environments since we deploy differently from how services are run locally

## Support services setup & management

Support services are documented in [System Architecture](./system-architecture.md). In summary, these are:

1. MongoDB for audit logging
2. MySQL for data persistence
3. Redis for a cache mecahnism
4. NATS for a queue mechanism

See [Data Stores](./system-architecture.md#data-stores) for more information on required tooling around these services.

### Starting/stoppping the support services

We orchestrate this via Docker Compose, a convenience script is provided at:

```sh
# bringing the services up
make compose_up

# bringing the services down
make compose_down
```

### Verifying support services

The Opsicle CLI tool comes up with a command that can verify support services are up

```sh
go run . utils check audit-database
go run . utils check cache
go run . utils check database
go run . utils check queue
```

### Scripts for support services

#### Migrating the database

```sh
make mysql_migrate;
```

#### Resetting the database

```sh
make mysql_reset;
```

## Start Opsicle services

1. Start the Approver service
   ```sh
   go run . start approver
   ```
   1. Use `--telegram-enabled` to enable Telegram bot (see [Setting up a Telegram bot](#setting-up-a-telegram-bot) for how to get the Telegram token)
   2. Use `--slack-enabled` to enable the Slack bot (see [Setting up a Slack bot](#setting-up-a-slack-bot) for how to get the Slack credentials)
2. Verify that the approver serivce is running at `http://localhost:13370`
   ```sh
   nc -zv 127.0.0.1 13370;
   ```
3. Start the Controller service
   ```sh
   go run . start controller
   ```
4. Verify that the controller serivce is running at `http://localhost:13371`
   ```sh
   nc -zv 127.0.0.1 13371;
   ```
5. Login as the user:
   ```sh
   opsicle login;
   ```


## Integrations

### Setting up a Slack bot

> This step is required in order to run the `approver` service with Slack approval channels enabled

1. Go to https://api.slack.com/apps and use the **Create New App** button to create a new application
2. Give it a name and enter other metadata type of details
3. Navigate to **OAuth & Permissions** and copy the **Bot User OAuthToken**, this will be `${SLACK_BOT_TOKEN}`
4. Still under **OAuth & Permissions**, add the following **Bot Token Scopes**:
   1. `app_mentions:read`
   2. `channels:read`
   2. `chat:write`
   2. `chat:write.public`
   2. `commands`
   2. `groups:read`
   2. `im:write`
   2. `users:read`
5. Navigate to **Interactivity & Shortcuts** and enable Interactivity
6. Navigate to **Socket Mode** and check the **Enable Socket Mode** toggle
7. An App Token will be shown, this will be `${SLACK_APP_TOKEN}`

A full app manifest is as follows:

```yaml
display_information:
  name: Opsicle Approval Service
features:
  bot_user:
    display_name: Opsicle Approval Service
    always_online: false
oauth_config:
  scopes:
    bot:
      - app_mentions:read
      - channels:read
      - chat:write
      - commands
      - groups:read
      - im:write
      - users:read
      - chat:write.public
settings:
  interactivity:
    is_enabled: true
  org_deploy_enabled: false
  socket_mode_enabled: true
  token_rotation_enabled: false
```

### Setting up a Telegram bot

> This step is required in order to run the `approver` service with Telegram approval channels enabled

1. Talk to the [https://t.me/BotFather](`@BotFather`) on Telegram
2. Create a new bot, give it an appropriate name
3. You will receive a token, save this as `${TELEGRAM_BOT_TOKEN}`
   - Optionally include it in the `.envrc` in the root of this repository
