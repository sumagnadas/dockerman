package cmd

import (
	"dock/utils"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
)

var ps_cmd = &cobra.Command{
	Use:   "ps",
	Short: "Get a list of running containers",
	Run:   psFunc,
}

func init() {
	root_cmd.AddCommand(ps_cmd)
}

func psFunc(c *cobra.Command, args []string) {
	// get containers from backend
	resp, err_conts := http.Get("http://localhost:4033/containers")
	if err_conts != nil {
		fmt.Println("daemon down or inaccessible.", err_conts)
		return
	}
	body, err_read := io.ReadAll(resp.Body)
	if err_read != nil {
		fmt.Println("Couldn't read body.", err_read)
		return
	}

	// read containers list from response
	var conts = []utils.ContState{}
	err_json := json.Unmarshal(body, &conts)
	if err_json != nil {
		fmt.Println("Not exactly json returned from daemon", err_json)
		return
	}

	// print containers list
	fmt.Println("Name\t\tImage\tNprocs\tRooted\t\tState")
	for _, cont := range conts {
		// State strings
		state := "Frozen"
		if cont.Running {
			state = "Running"
		}
		root_state := "Rootless"
		if cont.Rooted {
			root_state = "Rooted"
		}
		fmt.Printf("%s\t%s\t%d\t%s\t%s\n", cont.Name, cont.Image, cont.Nprocs, root_state, state)
	}
}
