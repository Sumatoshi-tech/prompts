// Package main provides the promptkit CLI entry point.
package main

import (
	"os"

	"github.com/Sumatoshi-tech/promptkit/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
