package main

import (
	"os"

	"github.com/telhawk-systems/telhawk-stack/cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
