// Command servitor switches managed blocks in files between configured states.
package main

import (
	"os"

	"github.com/nerdwave-nick/servitor/internal/cli"
)

func main() {
	os.Exit(cli.Execute(os.Args[1:], os.Stdout, os.Stderr))
}
