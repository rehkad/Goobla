package runner

import (
	"github.com/goobla/goobla/runner/gooblarunner"
	"github.com/goobla/goobla/runner/llamarunner"
)

func Execute(args []string) error {
	if args[0] == "runner" {
		args = args[1:]
	}

	var newRunner bool
	if args[0] == "--goobla-engine" {
		args = args[1:]
		newRunner = true
	}

	if newRunner {
		return gooblarunner.Execute(args)
	} else {
		return llamarunner.Execute(args)
	}
}
