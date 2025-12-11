# AGENTS.md

# General Conventions

## Naming Conventions

- Use the `New` verb for initialisation commands
- Components implementing a gracefully shutdown should include a `Shutdown` method

---

# Application Conventions

## Authentication and Authorisation

1. Use HTTP header `Authorization` for auth related to users
2. Use HTTP header `X-Api-Key` for internal use

---

# Cloud Infrastructure Conventions

---

# Networking Conventions

## Ports

- The `approver` service uses port `13370`
- The `controller` service uses port `13371`
- The `coordinator` service uses port `13372`
- The `worker` service uses port `13373`
- The `reporter` service uses port `13374`
