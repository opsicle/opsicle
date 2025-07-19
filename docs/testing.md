# Testing

No unit tests are done, you'll have to rely on these manual tests to verify that everything's working as expected.

## Approver Service

### Server configuration

#### Configure listening address

```sh
go run . start approver --listen-addr 127.0.0.1
```

The above starts a service listening only on `127.0.0.1`.

#### Enabling BasicAuth authentication

```sh
go run . start approver --basic-auth-enabled --basic-auth-username "user" --basic-auth-password "password"
```

The above requires the user `user` with password `password` to be specified as authentication credentials to any request to the server.

#### Enabling Bearer authentication

```sh
go run . start approver --bearer-auth-enabled --bearer-auth-token "8e11c462-6456-11f0-975f-fbc8b3a0068c"
```

The above requires the `Authorization` header to be sent with the value set to `Bearer 8e11c462-6456-11f0-975f-fbc8b3a0068c` in order to access the server.

#### Enabling IP allowlisting

```sh
go run . start approver --ip-allowlist-enabled --ip-allowlist "192.168.0.0/16,1.2.3.4,8.8.8.8"
```

The above example allows requests originating from the following CIDRs:
- `192.168.0.0/16`
- `1.2.3.4/32`
- `8.8.8.8/32`

### Slack approval flow

Start the approver service:

```sh
go run . start approver --slack-enabled
```

For the `--slack-enabled` to work, `${SLACK_BOT_TOKEN}` and `${SLACK_APP_TOKEN}` must be defined, see [./integrations.md](the integrations README) for instructions

#### Simple flow on Slack

```sh
go run . run approval ./examples/slack-wo-mfa.yaml
```

#### MFA flow on Slack

```sh
go run . run approval ./examples/slack-w-mfa.yaml
```

### Telegram approval flow

Start the approver service:

```sh
go run . start approver --telegram-enabled
```

For the `--telegram-enabled` to work, `${TELEGRAM_BOT_TOKEN}` must be defined, see [./integrations.md](the integrations README) for instructions

#### Simple flow on Telegram

```sh
go run . run approval ./examples/tg-wo-mfa.yaml
```

#### MFA flow on Telegram

```sh
go run . run approval ./examples/tg-w-mfa.yaml
```

### Webhook callback

```sh
go run . run approval ./examples/approval/tg-w-webhook-cb.yaml
```
