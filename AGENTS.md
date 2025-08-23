# AGENTS.md (global)

## Code style (universal)
- Langauges: use only Go for software code, use `/bin/sh`-compatible Shell for all scripts
- Third-party libraries: keep minimal, use the standard library as far as possible, for all dependencies, ensure minimal sub-dependencies and ensure it was last updated less than a year ago and has multiple contributors
- Comments: concise, follow best practices for Go, no fillter

## File layout
- `./cmd`: contains commands, each directory represents a command/sub-command
- `./pkg`: contains mainly SDKs for other apps to consume
- `./internal`: contains internal controller code
- `./deploy`: contains deployment manifests for deploying to Kubernetes and to a VM

## Go conventions
- Target Go: 1.24+
- Use `got fmt` and `go vet`, no PR if `go vet` fails
- This project uses `spf13/cobra` for structuring commands
- Commands go into `/cmd/*`; the root command package is contained in a directory in `./cmd`, with each subfolder being a sub-command
- SDKs for other Go apps to use is in `./pkg`
- Internal controllers are in `./internal`
- 
## Struct tags
- Always include both `json` and `yaml` tags in **camelCase** on exported fields.
- Example:

```go
type ExampleStruct struct {
  SampleBoolean bool `json:"sampleBoolean" yaml:"sampleBoolean"`
  SampleFloat float64 `json:"sampleFloat" yaml:"sampleFloat"`
  SampleNumber int64 `json:"sampleNumber" yaml:"sampleNumber"`
  SampleString string `json:"sampleString" yaml:"sampleString"`
}
```

## General API guidelines
- All resource manifests meant for ingestion by the system should be acceptable in both YAML and JSON formats
- Users can submit YAML manifests or send an API call with JSON in its body, use the `Content-Type` in the request to decide which to parse

## When told to add/create a CLI command
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
- For all CLI commands, add configuration by using the `Flags` from the package at `opsicle/internal/cli`, these should come at the top of the file
- Example initialisation of flags:
  ```go
  var flags cli.Flags = cli.Flags{
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

## Interactive text inputs in CLI
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

## Interactive selection from a list in CLI
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

## When told to add a controller endpoint
- Add the endpoint handler and routing in the appropriate file at `./internal/controller`, create a new `.go` file if necessary
- Add functions that handle state changes into the `models` package which will be called from the endpoint handler in `./internal/controller`
- Add an SDK method in `./pkg/controller` to call the endpoint

## Security guidelines
- Do not hardcode secrets
- All network calls must specify a timeout

## Testing
- Use `stretchr/testify` structures for testing

## Commit style
- Conventional commits: `feat:`, `fix:`, `docs:`, etc.
