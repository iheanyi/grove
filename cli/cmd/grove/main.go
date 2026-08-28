package main

import (
	"os"

	"github.com/iheanyi/grove/cli/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
