# Integrations

## Slack

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
