package main

import (
	"fmt"
	"os"

	"github.com/jhonxue/agent-fs/cmd"
	"github.com/jhonxue/agent-fs/pkg/apperr"
	"github.com/jhonxue/agent-fs/pkg/output"
)

// Version information (set via ldflags during build)
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

func main() {
	// Set version info in cmd package
	cmd.SetVersion(Version, Commit, BuildDate)

	if err := cmd.Execute(); err != nil {
		action, code, message := apperr.Parse(err, `command_execute`)
		if writeErr := output.PrintFailure(action, code, message); writeErr != nil {
			fmt.Fprintln(os.Stderr, writeErr.Error())
		}
		os.Exit(1)
	}
}
