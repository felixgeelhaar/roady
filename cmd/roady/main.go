package main

import (
	"os"

	"github.com/felixgeelhaar/roady/internal/infrastructure/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
