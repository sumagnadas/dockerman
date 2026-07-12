package cmd

import (
	"dock/utils"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
)

var psCmd = &cobra.Command{
	Use:   "ps",
	Short: "Get a list of running containers",
	Run:   ps_cont,
}

func init() {
	rootCmd.AddCommand(psCmd)
}

func ps_cont(c *cobra.Command, args []string) {
	resp, err := http.Get("http://localhost:4033/containers")
	if err != nil {
		fmt.Println("daemon down or inaccessible.", err)
		return
	}
	body, errRead := io.ReadAll(resp.Body)
	if errRead != nil {
		fmt.Println("Couldn't read body.", errRead)
		return
	}
	var conts = []utils.ContState{}
	errJson := json.Unmarshal(body, &conts)
	if errJson != nil {
		fmt.Println("Not exactly json returned from daemon", errJson)
		return
	}
	fmt.Println("Name\t\tImage\tNo. of starting procs.")
	for _, cont := range conts {
		fmt.Printf("%s\t%s\t%d\n", cont.Name, cont.Image, cont.Nprocs)
	}
}
