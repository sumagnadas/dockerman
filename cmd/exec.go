package cmd

import (
	"dock/utils"
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/spf13/cobra"
)

var exec_cmd = &cobra.Command{
	Use:   "exec [container_name] -- <command>",
	Short: "Execute a command inside a container.",
	Run:   executeFunc,
}

func init() {
	root_cmd.AddCommand(exec_cmd)
}

func executeFunc(c *cobra.Command, args []string) {
	if len(args) < 2 {
		fmt.Println("Not enough arguments.")
		fmt.Println(c.Use)
		return
	}
	name := args[0]

	// get container
	cont, err_get := utils.GetCont(name)
	if err_get != nil {
		fmt.Println("Couldn't get container:", err_get)
		return
	}
	// enter and execute command in container
	target_pid := cont.Procs[0]
	ns_args := []string{"-t", strconv.Itoa(target_pid), "--all"}
	nscmd := exec.Command("nsenter", append(ns_args, args[1:]...)...)

	// attach terminal I/O to container
	nscmd.Stdin = os.Stdin
	nscmd.Stderr = os.Stderr
	nscmd.Stdout = os.Stdout

	// start new command in container
	err_run := nscmd.Start()
	if err_run != nil {
		panic(err_run)
	}
	defer utils.WaitAndRemove(nscmd, cont.Name, nscmd.Process.Pid) // To make sure the golang CLI doesn't exit before the inner command attaches to the TTY

	// update container state
	cont.Procs = append(cont.Procs, nscmd.Process.Pid)
	cont.Nprocs += 1

	// update container in daemon
	err_upd := utils.UpdateCont(name, cont)
	if err_upd != nil {
		fmt.Println("Couldn't update container state:", err_upd)
	}
}
