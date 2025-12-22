# AGENTS.md

# General Guidelines

## Naming Conventions

- Use the `New` verb for initialisation commands
- Components implementing a gracefully shutdown should include a `Shutdown` method

## Runtime

- Use Go 1.20 and later
- When installing dependencies, run `go mod vendor` to pull dependencies in to the vendor directory

---

# Application

## Authentication and Authorisation

1. Use HTTP header `Authorization` for auth related to users
2. Use HTTP header `X-Api-Key` for internal use

## Cross-application communication

1. Opsicle services should use SDK packages located in `./pkg/*` for communicating between services. For example, when communicating with the Controller component from the CLI tool, the CLI tool should use an SDK method in `./pkg/controller` to make a call to the Controller service instead of implementing its own API call

## Structure

1. Each service is started using the command found at `./cmd/opsicle/start/${SERVICE_NAME}`
2. For each service, the internal mechanisms shall be located at the directory at `./internal/${SERVICE_NAME}`
3. Models of the service shall be located at the directory `./internal/${SERVICE_NAME}/models`

## How-Tos

### How to create a new method for ${SERVICE_NAME}

1. Create a HTTP handler function in an appropriate `./internal/${SERVICE_NAME}/*.go`. Create a new file if no appropriate file exists
2. Link it up with the route registration method so that it has an endpoint. Create the route registration method and add it to `./internal/${SERVICE_NAME}/http.go`
3. Create the SDK method at `./pkg/${SERVICE_NAME}/`

---

# Cloud Infrastructure Conventions

---

# Database

## Migrations

- Migration files are stored at `./internal/${COMPONENT}/migrations` depending on the component

## Table Conventions

- Always include an `id` field of `VARCHAR(36)` type that contains a UUID
- Always include a `created_at` field of `TIMESTAMP` type that should be the timestamp when a row is created (default to `NOW()` on row insertion)
- Always include a `last_updated_at` field of `TIMESTAMP` type that should be the timestamp when a row was last updated (default to `NOW()` on row insertion)

---

# Networking Conventions

## Ports

- The `approver` service uses port `13370`
- The `controller` service uses port `13371`
- The `coordinator` service uses port `13372`
- The `worker` service uses port `13373`
- The `reporter` service uses port `13374`
