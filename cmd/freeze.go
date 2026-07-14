package cmd

import (
	"dock/utils"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "freeze [container_name]",
	Short: "Freeze a running container",
	Run:   stop_cnt,
}

func init() {
	rootCmd.AddCommand(stopCmd)
}

func stop_cnt(c *cobra.Command, args []string) {
	if len(args) < 1 {
		fmt.Println(c.Use)
	}
	container := args[0]
	freezepath := filepath.Join("/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/dockerman", container, "cgroup.freeze")

	f, err := os.OpenFile(freezepath, os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Failed to open file: %s\n", err)
		return
	}
	defer f.Close()

	f.Write([]byte("1"))

	cont, err_get := utils.GetCont(container)
	if err_get != nil {
		fmt.Println("Couldn't get container", err)
		return
	}
	cont.Running = false
	err_upd := utils.UpdateCont(name, cont)
	if err_upd != nil {
		fmt.Println("Couldn't update container state.")
	}
}
