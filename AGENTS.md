# AGENTS.md (global)

## Styling and guidelines

### Code style (universal)
- Langauges: use only Go for software code, use `/bin/sh`-compatible Shell for all scripts
- Third-party libraries: keep minimal, use the standard library as far as possible, for all dependencies, ensure minimal sub-dependencies and ensure it was last updated less than a year ago and has multiple contributors
- Comments: concise, follow best practices for Go, no fillter
- Naming: use `cameCase` for all JSON and YAML struct tags
- When instantiating slices, do not use `make(T, 0)`, use the `[]T{}` semantics for standardisation
- When documenting APIs use comment notation from `swaggo`

#### Actions nomenclature
- As far as possible, use the following verbs for any function that manipulates/reads data:
  - Create*: for functions involving creation of a resource
  - Get*: for functions involving retrieval of a single resource
  - List*: for functions involving listing of multiple resources of a single resource type
  - Update*: for functions involving updating of details of a given resource type
  - Delete*: for functions involving removal/deletion of a given resource type
  - Execute*: for functions involving executing of a given resource type (where applicable, not every resource type is executable)
- Also use the above pattern for implementing any permissions involving actions

#### Go conventions
- Target Go: 1.24+
- Use `go fmt` and `go vet`, no PR if `go vet` fails
- Use `gorilla/mux` for creation of any HTTP-based servers
- Use `spf13/cobra` for structuring commands
- Use `spf13/viper` for managing the configuration
- Use `charmbracelet/bubbletea` for all interactive CLI UI components
- Use `sirupsen/logrus` for logging in the CLI
- Commands go into `/cmd/*`; the root command package is contained in a directory in `./cmd`, with each subfolder being a sub-command
- SDKs for other Go apps to use is in `./pkg`
- Internal controllers are in `./internal`
- Do not use inline structs, always define structs outside of the function conusming it

#### Error messages
- All errors should be in americaniZed english, meaning use 'z' over 's' in examples like 'authorized', 'unauthorized'

#### Struct tags
- Always include both `json` and `yaml` tags in **camelCase** on exported fields that are used by the SDK.
- Example:
  ```go
  type ExampleStruct struct {
    SampleBoolean bool `json:"sampleBoolean" yaml:"sampleBoolean"`
    SampleFloat float64 `json:"sampleFloat" yaml:"sampleFloat"`
    SampleNumber int64 `json:"sampleNumber" yaml:"sampleNumber"`
    SampleString string `json:"sampleString" yaml:"sampleString"`
  }
  ```

## Architecture guidelines
- Adopt an MVC code architecture
- For Models components, adopt an object-oriented approach where structs represent classes that have functions that manipualte an instance's properties

### Data persistence (database) preferences and guidelines
- Migrations can be found at `./internal/database/migrations` and uses `gomigrate`
- The CLI invocation that runs migrations can be found at `./cmd/opsicle/run/migrations`
- When creating migrations, use a timestamp (reference existing migrations) followed by a descriptive text in accordance with how `gomigrate` creates new migrations
- All database tables should include an `id` field which accounts for a UUID
- All database tables should include a `created_at` field which is set to the timestamp by default
- All `name` fields meant for storing human-readable names for a resource should be set to type `VARCHAR(255)` to accomodate 255 characters in a name
- Where applicable, the table should include a `created_by` field which is a UUID that references the `users`.`id` column
- All database tables should include a `last_updated_at` field which is set to the timestamp by default and is updated to the current timestamp of any update done on the row
- Where applicable, the table should include a `last_updated_by` field which is a UUID that references the `users`.`id` column

### API communications guidelines
- All resource manifests meant for ingestion by the system should be acceptable in both YAML and JSON formats
- Users can submit YAML manifests or send an API call with JSON in its body, use the `Content-Type` in the request to decide which to parse
- Exported SDKs (in `./pkg/`) should reference the internal controller's (as defined in MVC) types for contract parity
- For communication via RESTful APIs, implement a retry strategy that:
  - Only retries when it receives a HTTP status that definitely means the request didn't go through (IE when it's more than 500)
  - Has an exponential backoff strategy with a predefined (can be a variable) maximum number of retries
  - Logs all retry attempts
- For communication via gRPC streams:
  - Implement authentication via SSL certificates
  - Ensure that TLS can be established and that the certificate is valid
  - Do not implement mTLS (do not verify authenticity of certificate), only implement TLS to ensure data-in-transit is encrypted
  - Identify organisations via the Organization field of the SSL certificate which should be the organisation ID
  - Identify organisations via the CommonName field of the SSL certificate which should be the organisation codeword
  - Authenticate organisations via the OrganizationalUnit field of the SSL certificate which should an organisation API key

## Opsicle platform components
- This repository contains code to build a single binary that can be used to start 4 different components
- Commands to start the different components can be found in `./cmd/opsicle/start`
- The system components are:
  1. `approver`: The approver service which accepts requests from the `coordinator` service to trigger approval requests
  2. `controller`: The controller service which is the API server for any client tools (including the Opsicle CLI) to use to interact with the Opsicle platform
  3. `coordinator`: A service that retrieves pending automations from the NATS queue, handles the approvals if any, and submits it to the `worker` component for execution. Collects logs and execution information from the `worker` component when execution is completed.
  4. `worker`: A background service that connects via gRPC streams to the `coordinator` and receives automations to execute. Executes the automations as sent to it by the `coordinator` and returns data such as logs to the `coordinator`

### Networking and communication guidelines between components
- `approver` and `controller` interact via single-transaction RESTful API calls. Implement an appropriate retry strategy
- `controller` and `coordinator` interact via single-transaction RESTful API calls.
- `coordinator` and `worker` interact via gRPC streams.

### Controller component
- `./internal/database/migrations` contains the database migrations and consequently the database structure
- `./internal/controller` contains the Controller (as it is in MVC) 

## File layout
- `./bin`: contains built binaries and is generally kept empty for the VCS
- `./cmd`: contains commands, each directory represents a command/sub-command
- `./data`: contains data directories linked to local deployments (eg. those found in the `docker-compose.yml` in the root of this repository)
- `./deploy`: contains deployment manifests for deploying to Kubernetes and to a VM
- `./examples`: contains example manifests for Template, Automation, Approval resources written in YAML
- `./internal`: contains internal controller code
- `./pkg`: contains mainly Opsicle SDKs for other apps to consume; within Opsicle, if a service is consuming another service's functionality, it must use the SDK here


## Data persistence/storage implementation notes
Local deployments of these can be found in the `docker-compose.yml` at the root of this repository:
- MySQL is used for the platform database. All persistent data should be sent here
- Use MySQL workbench to manage the MySQL instance
- MongoDB is used for the audit database. All data related to audit logs should be sent here
- Use Mongo Compass to manage the MongoDB instance
- Redis is used as a system-wide cache. All pending transactions should be stored here while pending confirmation/action by the user
- Redis Insight is used as a visual tool for managing Redis
- NATS is used as a system-wide queue/streaming service.
- NATS-UI is used as a visual tool for managing NATS

## Infrastructure guidelines
- All infrastructure-as-code is to be written in Terraform
- Use Digital Ocean as the service provider


## When told to add things

### Adding functionality
- When told to add functionality, it usually involves
  - Adding/creating a CLI command to achieve the functionality
  - If involving the `controller` component
    - Writing of a SDK function in the `./pkg/controller` package
    - Writing of a handler in `./internal/controller` package
    - Writing of a model in `./internal/controller/models` package
    - The handler should call the model for all persistent data updates
  - If involving the `coordinator` component
    - Writing of a SDK function in the `./pkg/coordinator` package
    - Writing of a handler in `./internal/coordinator` package
    - Writing of a model in `./internal/coordinator/models` package
    - The handler should call the model for all persistent data updates
  - If involving the `worker` component
    - Writing of a SDK function in the `./pkg/worker` package
    - Writing of a handler in `./internal/worker` package
    - Writing of a model in `./internal/worker/models` package
    - The handler should call the model for all persistent data updates

### Adding/creating a CLI command
- The CLI tool is contained in `./cmd/opsicle/` as the root
- Each sub-directory in `./cmd/opsicle` represents a sub-command of the directory it is in
- To add a CLI command `opsicle create something great`, create a folder at `./cmd/opsicle/create/something/great`. Each directory should have a `.go` file of the same name and have a placeholder function.
- Example of filler subcommands before final command:
  ```go
  func init() {
    Command.Add(/* add the child sub-command's exported Command here */)
  }

  var Command = &cobra.Command{
    Use:     "get",
    Aliases: []string{"g"},
    Short:   "Retrieves resources in Opsicle",
    RunE: func(cmd *cobra.Command, args []string) error {
      return cmd.Help()
    },
  }
  ```
- For all CLI commands using flags, add configuration by using the `Flags` from the package at `opsicle/internal/cli`, these should come at the top of the file
- Example initialisation of flags:
  ```go
  var Flags cli.Flags = cli.Flags{
    {
      Name:         "sample-string-flag",
      DefaultValue: "",
      Usage:        "defines the value of a string",
      Type:         cli.FlagTypeString,
    },
    {
      Name:         "sample-integer-flag",
      DefaultValue: 587,
      Usage:        "defines the value of an integer",
      Type:         cli.FlagTypeInteger,
    },
    {
      Name:         "flag-that-enables-a-feature",
      DefaultValue: false,
      Usage:        "when this flag is specified, a feature is enabled",
      Type:         cli.FlagTypeBool,
    },
  }

  func init() {
    Flags.AddToCommand(Command)
  }
  ```
- For all CLI commands using flags, use the `BindViper` method of the `cli.Flags` object in the `PreRun` step in a Cobra command
- Example usage of `PreRun` to bind flags:
  ```go
  var Command = &cobra.Command{
    /* ... */
    PreRun: func(cmd *cobra.Command, args []string) {
      flags.BindViper(cmd)
    },
    /* ... */
  }
  ```
- For all CLI commands that involving Create / Read / Update/ Delete operations, these require a sign-in, so use the following to enforce authentication:
  ```go
  enforceAuth:
		sessionToken, err := cli.RequireAuth(controllerUrl, methodId)
		if err != nil {
			rootCmd := cmd.Root()
			rootCmd.SetArgs([]string{"login"})
			_, err := rootCmd.ExecuteC()
			if err != nil {
				return err
			}
			goto enforceAuth
		}
  ```
  The above block of code checks if a user's token is still valid and if it isn't, triggers a login attempt
- For displaying of data, check the global flag `output`
  - If it is `"json"`, print the JSON into the terminal via `os.Stdout`
  - If it is anything but `"json"` (classify this as `"test"` and use `fallthrough` to future-proof), use `tablewriter` to create a table and print it

### Adding an interactive simple text input in CLI
- Use the `CreatePrompt` method in the `internal/cli` package
- Example of a text/password input:
  ```go
  model := cli.CreatePrompt(cli.PromptOpts{
    Buttons: []cli.PromptButton{
      {
        Label: "Submit",
        Type:  cli.PromptButtonSubmit,
      },
      {
        Label: "Cancel / Ctrl + C",
        Type:  cli.PromptButtonCancel,
      },
    },
    Inputs: []cli.PromptInput{
      {
        Id:          "text-input",
        Placeholder: "Example text input as placeholder",
        Type:        cli.PromptString,
        Value:       defaultTextInputValue,
      },
      {
        Id:          "masked-text",
        Placeholder: "Masked text/passwords/codes",
        Type:        cli.PromptPassword,
        Value:       defaultMaskedTextInputValue,
      },
    },
  })
  prompt := tea.NewProgram(model)
  if _, err := prompt.Run(); err != nil {
    return fmt.Errorf("failed to get user input: %w", err)
  }
  if model.GetExitCode() == cli.PromptCancelled {
    return errors.New("user cancelled")
  }
  textInputValue := model.GetValue("text-input")
  maskedTextInputValue := model.GetValue("masked-text")
  ```

### Adding an interactive forms in CLI
- Example of a form input:
  ```go
  formFields := cli.FormFields{}

  // TODO: process the form fields as required

  variableInputForm := cli.CreateForm(cli.FormOpts{
    Title:       "title",
    Description: "Please enter/confirm values for the following",
    Fields:      formFields,
  })
  if err := variableInputForm.GetInitWarnings(); err != nil {
    return fmt.Errorf("failed to create form as expected: %w", err)
  }

  logrus.Debugf("executing form to get input from user")

  formProgram := tea.NewProgram(variableInputForm)
  formOutput, err := formProgram.Run()
  if err != nil {
    return fmt.Errorf("failed to get user input: %w", err)
  }
  var ok bool
  variableInputForm, ok = formOutput.(*cli.FormModel)
  if !ok {
    return fmt.Errorf("failed to receive a cli.FormModel")
  }
  if errors.Is(variableInputForm.GetExitCode(), cli.ErrorUserCancelled) {
    fmt.Println("💬 Alrights, tell me again if...")
    return cli.ErrorUserCancelled
  }
  inputVariableMap = variableInputForm.GetValueMap()
  ```


### Adding an interactive list selection in CLI
- Use the `CreateSelector` method in the `internal/cli` package for selection from a list of available values
- Example of a list selection input:
  ```go
  selectorChoices := []cli.SelectorChoice{}
  for _, option := range someData {
    selectorChoices = append(selectorChoices, cli.SelectorChoice{
      Description: option.Description, // this is displayed as "help text" to the user
      Label:       option.Label, // this is displayed to the user
      Value:       option.Value, // this is the returned value
    })
  }
  exampleSelector := cli.CreateSelector(cli.SelectorOpts{
    Choices: selectorChoices,
  })
  selector := tea.NewProgram(exampleSelector)
  if _, err := selector.Run(); err != nil {
    return fmt.Errorf("failed to get user input: %w", err)
  }
  if exampleSelector.GetExitCode() == cli.PromptCancelled {
    return errors.New("user cancelled")
  }
  selectedValue := exampleSelector.GetValue()
  ```

### Adding a `controller` endpoint
- Add the endpoint handler and routing in the appropriate file at `./internal/controller`, create a new `.go` file if necessary
- Add functions that handle state changes into the `models` package which will be called from the endpoint handler in `./internal/controller`
- Add an SDK method in `./pkg/controller` to call the endpoint

## Security guidelines
- Do not hardcode secrets

### Network security
- All network calls must specify a timeout

### Application security
- Secrets should not be logged
- User emails should not be logged, log the user ID instead
- Log only IDs and not any data that can be used to identify a user or organisation

### Data security
- All database calls must be done with statement preparation.
- For all classes (or `struct` in Go) that contain sensitive information like passwords or certificates or any other secret (eg. API keys, session/security tokens), include a `.GetRedacted()` method that returns a new instance of the class with those fields set to `null` or the zero-value. All API methods returning the class must only return the redacted version of the class.

## Testing
- Use `stretchr/testify` structures for testing

## Commit style
- Conventional commits: `feat:`, `fix:`, `docs:`, etc.
