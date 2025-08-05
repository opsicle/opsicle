# Setting up

## Setting up local environment

1. Make a clone of `.envrc.sample` into `.envrc`
2. Fill up the values according to the values in the password manager

## Setting up a Slack bot

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

## Setting up a Telegram bot

1. Talk to the [https://t.me/BotFather](`@BotFather`) on Telegram
2. Create a new bot, give it an appropriate name
3. You will receive a token, save this as `${TELEGRAM_BOT_TOKEN}`
   - Optionally include it in the `.envrc` in the root of this repository

## Initialising Opsicle

5. Start the support services by running
   ```sh
   make compose_up;
   ```
6. Verify the database is running
   1. Verify that a MySQL database is available at `127.0.0.1:3306`
      ```sh
      nc -zv 127.0.0.1 3306;
      ```
   2. Verify that you can get a shell via `mysql`
      ```sh
      mysql -uopsicle -h127.0.0.1 -P3306 -ppassword opsicle;
      ```
7. Verify that a cache is running
   1. Verify that a Redis cache is available at `127.0.0.1:6379`
      ```sh
      redis-cli -h 127.0.0.1 -p 6379 --user opsicle --pass password -n 0;
      ```
   2. Verify that you can get a shell via 
8. Run `go run . start approver`
   1. Use `--telegram-enabled` to enable Telegram bot
   2. Use `--slack-enabled` to enable the Slack bot
9.  Verify that the approver serivce is running at `http://localhost:12345`
   ```sh
   nc -zv 127.0.0.1 12345;
   ```
10. Run `go run . start controller`
12. Verify that the controller serivce is running at `http://localhost:54321`
   ```sh
   nc -zv 127.0.0.1 54321;
   ```
13. Create the `root` organisation and superuser
   ```sh
   opsicle init controller;
   ```
14. Login as the user:
   ```sh
   opsicle login;
   ```
