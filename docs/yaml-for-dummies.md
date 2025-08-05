# Yet-Another-Markup-Lanauage: YAML

**YAML** is a human-readable data serialization format commonly used for configuration files, including runbooks in Opsicle. Its key features are simplicity, readability, and a focus on structured data.

This guide covers the essentials you need to write valid, predictable YAML.

---

## 1. Basic Structure: Key-Value Pairs

At its core, YAML consists of key-value pairs.

```yaml
name: deploy-service
enabled: true
replicas: 3
```

**Rules:**
- Keys must be followed by a colon `:`
- Values can be strings, numbers, booleans, or nested structures

---

## 2. Indentation Matters

Indentation defines structure. YAML uses **spaces only** (never tabs).

```yaml
metadata:
  name: backup-job
  labels:
    environment: production
    team: platform
```

Each level is usually indented with **two spaces**, though four spaces is also common; just be consistent.

---

## 3. Lists

Use a dash (`-`) followed by a space to define list items.

```yaml
hosts:
  - web01
  - web02
  - web03
```

Lists can also contain maps:

```yaml
jobs:
  - name: build
    script: build.sh
  - name: deploy
    script: deploy.sh
```

---

## 4. Strings

You can quote strings, but it’s optional unless:
- The string contains `:`, `#`, `{}`, or special characters
- You want to preserve leading/trailing whitespace

```yaml
plain: hello
quoted: "world"
multilineWithNewlines: |
  This is line one.
  This is line two.
multilineBut: >
  This is line one.
  This is line two.
```

For multiline strings, use `|` to preserve newlines, and `>` to fold them into spaces.

---

## 5. Booleans and Null

YAML understands basic types:

```yaml
enabled: true
debug: false
cache: null
```

Aliases: `yes/no`, `on/off`, `~` (also means `null`).

---

## 6. Comments

Use `#` to write comments:

```yaml
# This runbook is used for production deployments
env: production
```

---

## 7. Anchors and Aliases (Advanced)

You can reuse parts of the document with anchors (`&`) and aliases (`*`):

```yaml
defaults: &common
  retries: 3
  timeout: 30

job:
  <<: *common
  name: deploy
```

This copies the `defaults` block into `job`.

---

## 8. Common Pitfalls

- **Don’t use tabs**: YAML will flip-table.
- **Watch your colons**: Always add a space after `:` for your sanity.
- **Be consistent with indentation**: Don’t mix up the number of spaces, it either looks untidy or you'll get errors.
- **Mistaking strings for numbers**: Some fields accept numbers but as strings (eg. `3` vs `"3"`), check the documentation or schema!

---

## Summary

YAML is pretty straightforward once you learn the core patterns:

- Key-value maps
- Lists
- Indentation-based nesting
- Optional quotes, with multi-line support
- Basic types (booleans, nulls, numbers)

Opsicle uses YAML to define automations, approvals, triggers, and execution environments. Writing clean YAML means your automation will be predictable, maintainable, and safe to run.
