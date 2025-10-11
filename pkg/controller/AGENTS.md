# Controller SDK
- This directory contains the SDK for the Controller component
- The endpoint handlers for these functions are in `./internal/controller` relative to the project root

# Conventions
- Use a central `Client` for all requests to the Controller component
- Do not create any other type of Client
- Append the version to function names
  - Example 1: CanUserV1 represents V1 of the CanUser functionionality
  - Example 2: GetOrgV1 represents V1 of the GetOrg functionality
- When asked to create another version of the functionality, add a new function but change the version
  - Example 1: GetUserV2 is added as a method of the `Client` struct if GetUserV1 already exists
