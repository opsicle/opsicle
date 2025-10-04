package main

import (
	"bytes"
	"errors"
	"fmt"
	"opsicle/cmd/opsicle"
	"opsicle/internal/cli"
	"opsicle/pkg/controller"
	"os"

	"github.com/sirupsen/logrus"
)

func init() {
}

func main() {
	// Execute root
	if err := opsicle.Command.Execute(); err != nil {
		cli.ExitCode |= cli.ExitCodeError
		handleError(err)
	}
	os.Exit(cli.ExitCode)
}

func handleError(err error) {
	if err != nil {
		logrus.Debugf("exitting because of error:\n```\n%s\n```\n", err)
	}
	switch true {
	case errors.Is(err, controller.ErrorDatabaseIssue):
		cli.PrintBoxedErrorMessage(
			`Unfortunately something went wrong on our side, our team has likely been alerted and will handle it`,
		)
		cli.ExitCode |= cli.ExitCodeDbError
	case errors.Is(err, cli.ErrorClientUnavailable):
		message := bytes.Buffer{}
		switch true {
		case errors.Is(err, controller.ErrorInvalidInput):
			fmt.Fprint(&message, "Encountered errors while trying to create the controller client, check your parameters and try again")
		}
		cli.PrintBoxedErrorMessage(message.String())
		cli.ExitCode |= cli.ExitCodeInputError
	case errors.Is(err, cli.ErrorControllerUnavailable):
		message := bytes.Buffer{}
		fmt.Fprint(&message, "The controller instance you are trying to connect to does not seem accessible")
		switch true {
		case errors.Is(err, controller.ErrorConnectionRefused):
			fmt.Fprint(&message, " (couldn't reach the controller URL)\n\n")
			fmt.Fprint(&message, "Verify your controller URL specified at --controller-url is valid, ")
		case errors.Is(err, controller.ErrorConnectionTimedOut):
			fmt.Fprint(&message, " (timed out while trying to reach the controller URL)\n")
			fmt.Fprint(&message, "Verify that you aren't being blocked by a firewall, ")
		}
		fmt.Fprint(&message, "you can run the following to verify connectivity:\n\n")
		fmt.Fprint(&message, "```\n")
		fmt.Fprint(&message, "nc -zv ${CONTROLLER_SVC_URL} ${CONTROLLER_SVC_PORT}\n")
		fmt.Fprint(&message, "```\n")
		cli.PrintBoxedErrorMessage(
			message.String(),
		)
		cli.ExitCode |= cli.ExitCodeServiceUnavailable
	}
}
