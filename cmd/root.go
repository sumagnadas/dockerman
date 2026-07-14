package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var root_cmd = &cobra.Command{
	Use:   "dock",
	Short: "Minimal container management system",
}

func Execute() {
	if err := root_cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
