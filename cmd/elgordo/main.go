package main

import (
	"os"

	"github.com/Miguelecf/elgordo-ia/internal/cli"
)

var version = "dev"

func main() {
	os.Exit(cli.Run(os.Args[1:], cli.Dependencies{
		Version: version,
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	}))
}
