package main

import (
	"fmt"
	"os"

	"github.com/amar-jay/amartrade/cmd"
)

var (
	version = "dev"
	date    = "unknown"
)

func main() {
	if err := cmd.Execute(version, date); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
