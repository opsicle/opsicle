# Opsicle

Opsicle is a Runbhook Automation platform.

# Documentation

## For Contributors

### Quicklinks

- [./docs/changelog/README.md](Changelog)
- [./docs/integrations.md](Integrations)
- [./docs/ideas.md](Idea log)
- [./docs/system-architecture.md](System Architecture)
- [./docs/testing.md](Testing)

### Setting up locally

1. Make a clone of `.envrc.sample` into `.envrc`
2. Fill up the values according to the values in the password manager

## For Users

The following instructions assume a deployment where the deployment is accessible over `localhost` or `127.0.0.1`. You may need to modify the URLs to hit the correct server on the correct network relative to your workstation.

### Deploying Opsicle

### Initialising Opsicle

1. Verify that the approver serivce is running at `http://localhost:12345`
   ```sh
   nc -zv 127.0.0.1 12345;
   ```
2. Verify that the controller serivce is running at `http://localhost:54321`
   ```sh
   nc -zv 127.0.0.1 54321;
   ```
3. Verify the database is running
   1. Verify that a MySQL database is available at `127.0.0.1:3306`
      ```sh
      nc -zv 127.0.0.1 3306;
      ```
   2. Verify that you can get a shell via `mysql`
      ```sh
      mysql -uopsicle -h127.0.0.1 -P3306 -ppassword opsicle;
      ```
4. Verify that a cache is running
   1. Verify that a Redis cache is available at `127.0.0.1:6379`
      ```sh
      redis-cli -h 127.0.0.1 -p 6379 --user opsicle --pass password -n 0;
      ```
   2. Verify that you can get a shell via 
5. Create the `root` organisation and superuser
   ```sh
   opsicle init controller;
   ```
6. Login as the user:
   ```sh
   opsicle login;
   ```
