package cmd

import (
	"bytes"
	"dock/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
)

func init() {
	rootCmd.AddCommand(runCmd)
}

var runCmd = &cobra.Command{
	Use:   "run [flags] -- <command>",
	Short: "Run a container runtime with image and command (attaches the stdin, stdout and stderr of the command to shell)",
	Run:   run,
}
var detach bool
var name string

func init() {
	runCmd.Flags().BoolVarP(&detach, "detach", "d", false, "Detach the stdin of the running command ")
	runCmd.Flags().StringVar(&name, "name", "", "Name of the container")
}

// docker         run image <cmd>
// go run main.go run image <cmd>
func run(cmd *cobra.Command, args []string) {
	if len(args) < 2 {
		fmt.Println("Not enough arguments.")
		return
	}

	// divide the commandline arguments
	image := args[0]
	cmdline := args[1:]

	wd, _ := os.Getwd()
	img_path := filepath.Join(wd, image)

	// check if image exists
	if _, err := os.Stat(img_path); err != nil {
		fmt.Println("Image/root filesystem not found or inaccessible at ", img_path)
		return
	}

	// debug
	fmt.Printf("Running with image '%s' and command %v as %d. :)\n", image, cmdline, os.Getpid())

	if os.Getpid() == 1 {
		// We are officially inside the container...
		cmd := exec.Command(cmdline[0], cmdline[1:]...)

		if !detach {
			// link all the system FDs with the terminal FDs
			cmd.Stdin = os.Stdin
		}
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		hname := name
		if hname == "" {
			hname = "container"
		}
		// set hostname to differentiate
		unix.Sethostname([]byte(hname))

		// set root and mount proc
		fmt.Println("Changing root to ", img_path)
		unix.Mount(img_path, img_path, "none", unix.MS_BIND, "")

		// pivot root
		unix.Chdir(img_path)
		unix.PivotRoot(".", "old_root")
		unix.Mount("proc", "proc", "proc", 0, "")

		// detach old_root
		unix.Unmount("/old_root", unix.MNT_DETACH)

		// for now, make sure the filesystem is unmounted before exiting the container
		defer unix.Unmount("/proc", unix.MNT_DETACH)

		// run it
		errRun := cmd.Run()
		if errRun != nil {
			panic(errRun)
		}
	} else if len(cmdline) != 0 {
		if os.Getuid() == 0 {
			// set up the other namespaces as the host with root user (in semi-container)
			cmd := exec.Command("/proc/self/exe", os.Args[1:]...)

			if !detach {
				// link all the system FDs with the terminal FDs
				cmd.Stdin = os.Stdin
				cmd.Stderr = os.Stderr
				cmd.Stdout = os.Stdout
			}

			// Namespaces
			cmd.SysProcAttr = &syscall.SysProcAttr{
				Cloneflags:   unix.CLONE_NEWUTS | unix.CLONE_NEWPID | unix.CLONE_NEWNET | unix.CLONE_NEWNS,
				Unshareflags: unix.CLONE_NEWNS, // unshare the mount namespace to not show any mounts from the container. it's shared by default.
			}

			// start the container runtime
			errRun := cmd.Start()
			if errRun != nil {
				panic(errRun)
			}

			// Add to the manager when a new container is opened
			if name == "" {
				newname, err := utils.GenerateRandomHash(8) // generate a name based on random hash
				if err != nil {
					name = "random1234"
				} else {
					name = newname
				}
			}
			cont := utils.ContState{
				Name:   name,
				Image:  image,
				Nprocs: 1,
				Procs:  []int{cmd.Process.Pid},
			}
			body, _ := json.Marshal(cont)
			_, err := http.Post("http://localhost:4033/add", "application/json", bytes.NewBuffer(body))
			if err != nil {
				fmt.Println("POST failed: ", err)
			}
			defer utils.WaitAndRemove(cmd, name, cmd.Process.Pid) // To make sure the golang CLI doesn't exit before the inner command attaches to the TTY

		} else {
			// set up the user namespace for container as the host user rootless
			cmd := exec.Command("/proc/self/exe", os.Args[1:]...)

			if !detach {
				// link all the system FDs with the terminal FDs
				cmd.Stdin = os.Stdin
			}
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			// Namespaces
			cmd.SysProcAttr = &syscall.SysProcAttr{
				Cloneflags: unix.CLONE_NEWUSER,
				UidMappings: []syscall.SysProcIDMap{
					{
						ContainerID: 0, HostID: 1000, Size: 1,
					},
				},
				GidMappings: []syscall.SysProcIDMap{
					{
						ContainerID: 0, HostID: 1000, Size: 1,
					},
				},
			}

			// start the container runtime
			errRun := cmd.Run()
			if errRun != nil {
				panic(errRun)
			}
		}
		fmt.Println("Container exited...")
	}
}
