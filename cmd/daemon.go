package cmd

import (
	"dock/utils"
	"fmt"
	"net/http"
	"strconv"

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

func removePid(cont *utils.ContState, pid int) {
	ind := -1
	for i, v := range cont.Procs {
		if v == pid {
			ind = i
			break // Stop searching after finding the first match
		}
	}

	// 2. Remove the element using slicing if found
	if ind != -1 {
		// The '...' unpacks the second slice to pass elements individually
		cont.Procs = append(cont.Procs[:ind], cont.Procs[ind+1:]...)
		cont.Nprocs -= 1
	}
}

func removeCont(ctx *gin.Context) {
	var ind int = -1
	name := ctx.Query("name")
	pid, errAtoi := strconv.Atoi(ctx.Query("pid"))

	if name == "" {
		ctx.IndentedJSON(400, "Need name query parameter")
		return
	}
	if errAtoi != nil {
		ctx.IndentedJSON(400, "Need proper pid query parameter")
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
	removePid(&containers[ind], pid)
	if containers[ind].Nprocs == 0 {
		containers = append(containers[:ind], containers[ind+1:]...)
	}
	ctx.IndentedJSON(http.StatusAccepted, "Deleted")
}
func getContainers(ctx *gin.Context) {
	ctx.IndentedJSON(http.StatusOK, containers)
}
func getContainer(ctx *gin.Context) {
	name := ctx.Query("name")
	if name == "" {
		return
	}
	for _, cont := range containers {
		if cont.Name == name {
			ctx.IndentedJSON(http.StatusOK, cont)
			return
		}
	}
	ctx.IndentedJSON(http.StatusInternalServerError, "Not Found")
}
func updateContainer(ctx *gin.Context) {
	var updCont utils.ContState

	if err := ctx.BindJSON(&updCont); err != nil {
		fmt.Println(err)
		return
	}

	for i, cont := range containers {
		if cont.Name == updCont.Name {
			containers[i] = updCont
			ctx.IndentedJSON(http.StatusAccepted, "Updated")

			return
		}
	}
	ctx.IndentedJSON(http.StatusInternalServerError, "Server error")
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
	router.GET("/remove", removeCont)
	router.GET("/containers", getContainers)
	router.GET("/get", getContainer)
	router.POST("/update", updateContainer)

	router.Run("localhost:4033")
}
