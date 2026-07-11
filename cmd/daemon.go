package cmd

import (
	"dock/utils"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
)

var containers = []utils.ContState{}

func addCont(ctx *gin.Context) {
	var newCont utils.ContState

	if err := ctx.BindJSON(&newCont); err != nil {
		fmt.Println(err)
		return
	}

	containers = append(containers, newCont)
	ctx.IndentedJSON(http.StatusAccepted, newCont)
}
func removeCont(ctx *gin.Context) {
	var ind int = -1
	var name string

	if err := ctx.ShouldBindPlain(&name); err != nil {
		fmt.Println(err)
		return
	}

	for i, cont := range containers {
		if cont.Name == name {
			ind = i
		}
	}
	if ind == -1 {
		return
	}
	if ind != len(containers)-1 {
		containers = append(containers[:ind], containers[ind+1:]...)
	} else {
		containers = containers[:ind]
	}
	ctx.IndentedJSON(http.StatusAccepted, "Deleted")
}
func getContainers(ctx *gin.Context) {
	ctx.IndentedJSON(http.StatusOK, containers)
}

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Launch a daemon to manage containers.",
	Run:   daem,
}

func init() {
	rootCmd.AddCommand(daemonCmd)
}

func daem(cmd *cobra.Command, args []string) {
	router := gin.Default()
	router.POST("/add", addCont)
	router.POST("/remove", removeCont)
	router.GET("/containers", getContainers)

	router.Run("localhost:4033")
}
