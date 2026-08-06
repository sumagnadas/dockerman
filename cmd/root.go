package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var root_cmd = &cobra.Command{
	Use:   "dockman",
	Short: "Minimal container management system",
}
var addr string

func init() {
	root_cmd.PersistentFlags().StringVar(&addr, "addr", "localhost:4033", "Address of the daemon")
}

func Execute() {
	if err := root_cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
