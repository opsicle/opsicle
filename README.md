# Opsicle

Opsicle is a Runbhook Automation platform.

# Documentation

## For Contributors

- [./docs/changelog/README.md](Changelog)
- [./docs/integrations.md](Integrations)
- [./docs/ideas.md](Idea log)
- [./docs/system-architecture.md](System Architecture)
- [./docs/testing.md](Testing)

## For Users

The following instructions assume a deployment where the deployment is accessible over `localhost` or `127.0.0.1`. You may need to modify the URLs to hit the correct server on the correct network relative to your workstation.

### Deploying Opsicle

### Initialising Opsicle

1. Verify that the approver serivce is running at `http://localhost:12345`
1. Verify that the controller serivce is running at `http://localhost:54321`
   ```sh

   ```
2. Verify the database is running
   1. Verify that a MySQL database is available at `127.0.0.1:3306`
      ```sh
      nc -zv 127.0.0.1 3306
      ```
3. Verify that a cache is running
   1. Verify that a Redis cache is available at `127.0.0.1:6379`
      ```sh
      nc -zv 127.0.0.1 6379
      ```
1. Create the `root` organisation and superuser
   ```sh
   opsicle init controller;
   ```
1. Login as the user:
   ```sh
   opsicle login;
   ```
