package main

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/moogla/moogla/cmd"
)

func main() {
	cobra.CheckErr(cmd.NewCLI().ExecuteContext(context.Background()))
}
