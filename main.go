package main

import (
	"errors"
	"fmt"
	"opsicle/cmd/opsicle"
	"opsicle/pkg/controller"
	"os"

	"github.com/sirupsen/logrus"
)

func init() {
}

func main() {
	// Execute root
	if err := opsicle.Command.Execute(); err != nil {
		handleError(err)
		os.Exit(1)
	}
}

func handleError(err error) {
	if err != nil {
		logrus.Errorf("Exiting due to error: %s\n", err)
	}
	switch true {
	case errors.Is(err, controller.ErrorConnectionRefused):
		fmt.Println("⚠️  The controller instance you are trying to connect to does not seem accessible")
		fmt.Println("   > Verify your controller URL specified at --controller-url")
		fmt.Println("   > You can run the following to verify connectivity:")
		fmt.Println("     ```")
		fmt.Println("     nc -zv ${CONTROLLER_SVC_URL} ${CONTROLLER_SVC_PORT}")
		fmt.Println("     ```")
		os.Exit(2)
	case errors.Is(err, controller.ErrorConnectionTimedOut):
		fmt.Println("⚠️  The controller instance you are trying to connect to has timed out on a connection")
		fmt.Println("   > Verify that you aren't being blocked by a firewall")
		fmt.Println("   > You can run the following to verify connectivity:")
		fmt.Println("     ```")
		fmt.Println("     nc -zv ${CONTROLLER_SVC_URL} ${CONTROLLER_SVC_PORT}")
		fmt.Println("     ```")
		os.Exit(2)
	}
}
