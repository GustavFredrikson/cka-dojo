// Command dojo is the CKA training CLI.
package main

import (
	"os"

	"github.com/gustavfredrikson/cka-dojo/internal/cli"
)

func main() { os.Exit(cli.Execute()) }
