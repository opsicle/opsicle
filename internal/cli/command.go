package cli

import (
	"errors"
	"opsicle/internal/common"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/spf13/cobra"
)

var commandProcessWaiter sync.WaitGroup

type CommandOpts struct {
	Name  string
	Flags Flags

	Use     string
	Aliases []string
	Short   string
	Long    string

	Run func(cmd *cobra.Command, opts *Command, args []string) error
}

// NewCommand initialises amd returns a data structure that contains
// a set of common constructs and information for all commands to use
func NewCommand(opts CommandOpts) *Command {
	output := &Command{
		name:              opts.Name,
		shutdownProcesses: map[string]func() error{},
	}
	serviceLogs := make(chan common.ServiceLog, 64)
	common.StartServiceLogLoop(serviceLogs)
	output.serviceLogs = &serviceLogs

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown_hostname"
	}
	output.hostname = hostname
	output.user = os.Getuid()
	output.group = os.Getgid()

	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}
	output.workingDirectory = wd
	output.Command = &cobra.Command{
		Use:     opts.Use,
		Aliases: opts.Aliases,
		Short:   opts.Short,
		Long:    opts.Long,
		PreRun: func(cmd *cobra.Command, args []string) {
			opts.Flags.BindViper(cmd)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			err := opts.Run(cmd, output, args)
			if !output.isShutdownFromSignal {
				output.Shutdown()
				commandProcessWaiter.Done()
			}
			commandProcessWaiter.Wait()
			return err
		},
	}
	opts.Flags.AddToCommand(output.Command)

	return output
}

// Command is an abstraction for all of opsicle cli's commands
type Command struct {
	errs                 []error
	flags                Flags
	name                 string
	user                 int
	group                int
	hostname             string
	isShutdownFromSignal bool
	serviceLogs          *chan common.ServiceLog
	shutdownProcesses    map[string]func() error
	workingDirectory     string

	*cobra.Command
}

// AddShutdownProcess adds a `process` named `id` for use when the
// Shutdown() method is called
func (cd *Command) AddShutdownProcess(id string, process func() error) {
	if _, ok := cd.shutdownProcesses[id]; ok {
		*cd.serviceLogs <- common.ServiceLogf(common.LogLevelWarn, "process[%s] was overwritten")
	}
	cd.shutdownProcesses[id] = process
}

// Error returns any errors
func (cd *Command) Error() error {
	return errors.Join(cd.errs...)
}

// Get returns the underlying cobra.Command (or whatever is used in future
// to implement the Command interface) instance
func (cd *Command) Get() *cobra.Command {
	return cd.Command
}

// GetFlags returns the flagset of this command, useful when creating
// alternate names for commands and needing to replicate the flagset
func (cd *Command) GetFlags() Flags {
	return cd.flags
}

// GetFullanme returns the full namespaced ID of the current command
func (cd *Command) GetFullname() string {
	return strings.ToLower("opsicle." + cd.name)
}

// GetFullanme returns the full namespaced ID of the current command
func (cd *Command) GetSnakeCaseName() string {
	return strings.ReplaceAll(strings.ToLower("opsicle."+cd.name), ".", "_")
}

// GetGroupId returns the group ID of the user running the application,
// useful when debugging permission errors if any
func (cd *Command) GetGroupId() int {
	return cd.group
}

// GetHostname returns the current hostname of the machine, useful
// for identifying connections and node issues especially in the case
// of flaky network errors
func (cd *Command) GetHostname() string {
	return cd.hostname
}

// GetServiceLogs returns an instance of the service logs channel
// that other components can use for logging to a central logging
// system
func (cd *Command) GetServiceLogs() chan common.ServiceLog {
	return *cd.serviceLogs
}

// GetUserId retrieves the ID of the user running the application,
// useful when debugging permission errors if any
func (cd *Command) GetUserId() int {
	return cd.user
}

// GetWorkingDirectory retrieves the current working directory
func (cd *Command) GetWorkingDirectory() string {
	return cd.workingDirectory
}

// IsReady tells the command to begin listening for system lifecycle
// events
func (cd *Command) IsReady() {
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM)
	var isShutdownFromSignalMutex sync.Mutex

	commandProcessWaiter.Add(1)
	go func() {
		<-signalChannel
		isShutdownFromSignalMutex.Lock()
		cd.isShutdownFromSignal = true
		isShutdownFromSignalMutex.Unlock()
		cd.Shutdown()
		commandProcessWaiter.Done()
	}()
}

// Shutdown gracefully terminates any processes in the command, for this
// to be effective, use the AddShutdownProcess method to add functions
// that closes things like database connections or terminates any
// long-running tasks (WHY THO) gracefully
func (cd *Command) Shutdown() {
	var waiter sync.WaitGroup
	*cd.serviceLogs <- common.ServiceLogf(common.LogLevelInfo, "triggering shutdownProcesses (%v registered)", len(cd.shutdownProcesses))
	succeededCount := 0
	failedCount := 0
	var countMutex sync.Mutex
	for id, shutdownProcess := range cd.shutdownProcesses {
		waiter.Add(1)
		go func(processId string) {
			defer waiter.Done()
			*cd.serviceLogs <- common.ServiceLogf(common.LogLevelInfo, "triggering shutdownProcess[%s]", processId)
			if err := shutdownProcess(); err != nil {
				*cd.serviceLogs <- common.ServiceLogf(common.LogLevelError, "shutdownProcess[%s] failed: %s", processId, err.Error())
				countMutex.Lock()
				failedCount++
				countMutex.Unlock()
				return
			}
			countMutex.Lock()
			succeededCount++
			countMutex.Unlock()
			*cd.serviceLogs <- common.ServiceLogf(common.LogLevelInfo, "shutdownProcess[%s] succeeded", processId)
		}(id)
	}
	waiter.Wait()
	*cd.serviceLogs <- common.ServiceLogf(common.LogLevelInfo, "completed shutdownProcesses (%v successful, %v errored out)", succeededCount, failedCount)
}
