package cmd

import (
	"dock/utils"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var unfreeze_cmd = &cobra.Command{
	Use:   "unfreeze [container_name]",
	Short: "Unfreeze a freezed container",
	Run:   unfreezeFunc,
}

func init() {
	root_cmd.AddCommand(unfreeze_cmd)
}

func unfreezeFunc(c *cobra.Command, args []string) {
	if len(args) < 1 {
		fmt.Println(c.Use)
		return
	}

	// unfreeze using cgroup
	container := args[0]
	freezepath := filepath.Join("/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/dockerman", container, "cgroup.freeze")

	f, err_open := os.OpenFile(freezepath, os.O_WRONLY, 0644)
	if err_open != nil {
		fmt.Printf("Failed to open file: %s\n", err_open)
		return
	}
	defer f.Close()

	f.Write([]byte("0"))

	// get container
	cont, err_get := utils.GetCont(container)
	if err_get != nil {
		fmt.Println("Couldn't get container:", err_get)
		return
	}

	// update container state
	cont.Running = true

	// update daemon
	err_upd := utils.UpdateCont(name, cont)
	if err_upd != nil {
		fmt.Println("Couldn't update container state:", err_upd)
	}
}
