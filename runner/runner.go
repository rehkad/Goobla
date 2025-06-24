package runner

import (
	"github.com/moogla/moogla/runner/llamarunner"
	"github.com/moogla/moogla/runner/mooglarunner"
)

func Execute(args []string) error {
	if args[0] == "runner" {
		args = args[1:]
	}

	var newRunner bool
	if args[0] == "--ollama-engine" {
		args = args[1:]
		newRunner = true
	}

	if newRunner {
		return mooglarunner.Execute(args)
	} else {
		return llamarunner.Execute(args)
	}
}
