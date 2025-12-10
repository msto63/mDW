package main

import (
	"os"

	"github.com/msto63/mDW/cmd/mdw/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
