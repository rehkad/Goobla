package main

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/goobla/goobla/cmd"
)

func main() {
	cobra.CheckErr(cmd.NewCLI().ExecuteContext(context.Background()))
}
