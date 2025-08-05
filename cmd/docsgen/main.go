package main

import (
	"fmt"
	"opsicle/cmd/docsgen/docsgen"
	"os"
)

func init() {
}

func main() {
	// Execute root
	if err := docsgen.Command.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
