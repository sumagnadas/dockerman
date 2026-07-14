package cmd

import (
	"bytes"
	"dock/utils"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

	resp, err := http.Get("http://localhost:4033/get?name=" + container)
	if err != nil {
		fmt.Println(err)
		return
	}
	if resp.StatusCode == 500 {
		fmt.Println("Container with name", container, "does not exist.")
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
	cont.Running = false
	upd_cont, _ := json.Marshal(cont)
	_, errUpd := http.Post("http://localhost:4033/update", "application/json", bytes.NewBuffer(upd_cont))
	if errUpd != nil {
		fmt.Println("Couldn't update container.", errUpd)
		return
	}
}
