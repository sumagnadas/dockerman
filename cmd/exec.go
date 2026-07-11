package cmd

import (
	"bytes"
	"dock/utils"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"

	"github.com/spf13/cobra"
)

var execCmd = &cobra.Command{
	Use:   "exec [container_name] -- <command>",
	Short: "Execute a command inside a container.",
	Run:   execute_cnt,
}

func init() {
	rootCmd.AddCommand(execCmd)
}

func execute_cnt(cmd *cobra.Command, args []string) {
	if len(args) < 2 {
		fmt.Println("Not enough arguments.")
		return
	}
	name := args[0]
	resp, err := http.Get("http://localhost:4033/get?name=" + name)
	if err != nil {
		fmt.Println(err)
		return
	}
	if resp.StatusCode == 500 {
		fmt.Println("Container with name", name, "does not exist.")
		return
	}

	body, errRead := io.ReadAll(resp.Body)
	if errRead != nil {
		fmt.Println("Couldn't read body.", errRead)
		return
	}
	var cont utils.ContState
	errJson := json.Unmarshal(body, &cont)
	if errJson != nil {
		fmt.Println("Not exactly json?", errJson)
		return
	}
	target_pid := cont.Procs[0]
	ns_args := []string{"-t", strconv.Itoa(target_pid), "--all"}
	nscmd := exec.Command("nsenter", append(ns_args, args[1:]...)...)

	nscmd.Stdin = os.Stdin
	nscmd.Stderr = os.Stderr
	nscmd.Stdout = os.Stdout
	errRun := nscmd.Start()
	if errRun != nil {
		panic(errRun)
	}
	cont.Procs = append(cont.Procs, nscmd.Process.Pid)
	cont.Nprocs += 1
	upd_cont, _ := json.Marshal(cont)
	_, errUpd := http.Post("http://localhost:4033/update", "application/json", bytes.NewBuffer(upd_cont))
	if errUpd != nil {
		fmt.Println("Couldn't update container.", errUpd)
		return
	}
	defer utils.WaitAndRemove(nscmd, cont.Name, nscmd.Process.Pid) // To make sure the golang CLI doesn't exit before the inner command attaches to the TTY

}
