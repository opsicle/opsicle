package main

import (
	"fmt"
	"opsicle/cmd/opsicle"
	"os"
)

func init() {
}

func main() {
	// Execute root
	if err := opsicle.Command.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
