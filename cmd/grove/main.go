package main

import (
	"os"

	"github.com/iheanyi/grove/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
