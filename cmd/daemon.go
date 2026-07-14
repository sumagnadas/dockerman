package cmd

import (
	"dock/utils"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
)

var containers = []utils.ContState{}

/*
POST /add

Adds a container to the list.
*/
func addCont(ctx *gin.Context) {
	var newCont utils.ContState

	if err := ctx.BindJSON(&newCont); err != nil {
		fmt.Println(err)
		return
	}

	containers = append(containers, newCont)
	ctx.IndentedJSON(http.StatusAccepted, newCont)
}

/* Helper function to remove PID from container process list */
func removePid(cont *utils.ContState, pid int) {
	ind := -1
	for i, v := range cont.Procs {
		if v == pid {
			ind = i
			break
		}
	}

	if ind != -1 {
		cont.Procs = append(cont.Procs[:ind], cont.Procs[ind+1:]...)
		cont.Nprocs -= 1
	}
}

/*
GET /remove?name=<name>&pid=<pid>

Removes PID <pid> from  process list of container <name>
Removes container if process list is empty
*/
func removeCont(ctx *gin.Context) {
	var ind int = -1
	name := ctx.Query("name")
	pid, err_atoi := strconv.Atoi(ctx.Query("pid"))

	// query validation
	if name == "" {
		ctx.IndentedJSON(400, "Need name query parameter")
		return
	}
	if err_atoi != nil {
		ctx.IndentedJSON(400, "Need proper pid query parameter")
		return
	}

	// find container using name from list (really need to make it a map)
	for i, cont := range containers {
		if cont.Name == name {
			ind = i
		}
	}
	if ind == -1 {
		return
	}

	// remove pid from container list
	removePid(&containers[ind], pid)

	// remove container if process list is empty
	if containers[ind].Nprocs == 0 {
		containers = append(containers[:ind], containers[ind+1:]...)

		// remove cgroup dir
		cgroup_dir := filepath.Join("/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/dockerman", name)
		os.Remove(cgroup_dir)
	}
	ctx.IndentedJSON(http.StatusAccepted, "Deleted")
}

/*
GET /containers

Returns list of container with relevant info as JSON
*/
func getContainers(ctx *gin.Context) {
	ctx.IndentedJSON(http.StatusOK, containers)
}

/*
GET /get?name=<name>

Returns info about a specific container as JSON.
*/
func getContainer(ctx *gin.Context) {
	name := ctx.Query("name")

	// query validation
	if name == "" {
		return
	}

	//find and return container
	for _, cont := range containers {
		if cont.Name == name {
			ctx.IndentedJSON(http.StatusOK, cont)
			return
		}
	}
	ctx.IndentedJSON(http.StatusInternalServerError, "Not Found")
}

/*
POST /update

Updates a container state, based on name
*/
func updateContainer(ctx *gin.Context) {
	var updCont utils.ContState

	// request validation
	if err := ctx.BindJSON(&updCont); err != nil {
		fmt.Println(err)
		ctx.IndentedJSON(http.StatusInternalServerError, "Not proper request body")
		return
	}

	// find and update container
	for i, cont := range containers {
		if cont.Name == updCont.Name {
			containers[i] = updCont
			ctx.IndentedJSON(http.StatusAccepted, "Updated")

			return
		}
	}
	ctx.IndentedJSON(http.StatusInternalServerError, "Server couldn't find container with name "+updCont.Name)
}

var daemon_cmd = &cobra.Command{
	Use:   "daemon",
	Short: "Launch a daemon to manage containers.",
	Run:   daemonFunc,
}

func init() {
	root_cmd.AddCommand(daemon_cmd)
}

func daemonFunc(cmd *cobra.Command, args []string) {
	err_cgroupdir := os.Mkdir("sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/dockerman", 0755)
	if err_cgroupdir != nil {
		fmt.Println("Couldn't set up root cgroup dir", err_cgroupdir)
		return
	}

	router := gin.Default()
	router.POST("/add", addCont)
	router.GET("/remove", removeCont)
	router.GET("/containers", getContainers)
	router.GET("/get", getContainer)
	router.POST("/update", updateContainer)

	router.Run("localhost:4033")
}
