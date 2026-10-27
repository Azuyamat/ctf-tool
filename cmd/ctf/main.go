package main

import (
	"fmt"
	"os"

	"github.com/azuyamat/ctf/internal/commands"
)

func main() {
	if err := commands.Root().Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
