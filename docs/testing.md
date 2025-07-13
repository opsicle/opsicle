# Testing

No unit tests are done, you'll have to rely on these manual tests to verify that everything's working as expected.

## Approver Service

Start the approver service:

```sh
go run . start approver --telegram-enabled --slack-enabled
```

- For the `--telegram-enabled` to work, `${TELEGRAM_BOT_TOKEN}` must be defined, see [./integrations.md](the integrations README) for instructions
- For the `--slack-enabled` to work, `${SLACK_BOT_TOKEN}` and `${SLACK_APP_TOKEN}` must be defined, see [./integrations.md](the integrations README) for instructions

### Slack flows

#### Simple flow on Slack

```sh
go run . run approval ./examples/slack-wo-mfa.yaml
```

#### MFA flow on Slack

```sh
go run . run approval ./examples/slack-w-mfa.yaml
```

### Telegram flows

#### Simple flow on Telegram

```sh
go run . run approval ./examples/tg-wo-mfa.yaml
```

#### MFA flow on Telegram

```sh
go run . run approval ./examples/tg-w-mfa.yaml
```

### Webhook callback flow

For Telegram:

```sh
go run . run approval ./examples/approval/tg-w-webhook-cb.yaml
```
