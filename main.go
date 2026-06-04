package main

import (
	"os"

	"github.com/asam264/agentspace/cmd"
	"github.com/asam264/agentspace/internal/ui"
)

func main() {
	if err := cmd.Execute(); err != nil {
		ui.Error("%s", err.Error())
		os.Exit(1)
	}
}
