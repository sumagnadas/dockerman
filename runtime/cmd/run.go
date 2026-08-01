package cmd

import (
	"dock/utils"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
)

var run_cmd = &cobra.Command{
	Use:   "run image [flags] -- <command>",
	Short: "Run a container runtime with image and command (attaches the stdin, stdout and stderr of the command to shell)",
	Run:   runFunc,
}
var user int
var name string

func init() {
	root_cmd.AddCommand(run_cmd)
	run_cmd.Flags().IntVar(&user, "user", 0, "UID of the user")
	run_cmd.Flags().StringVar(&name, "name", "", "Name of the container")
}

// docker         run image <cmd>
// go run main.go run image <cmd>
func runFunc(c *cobra.Command, args []string) {
	if len(args) < 2 {
		fmt.Println("Not enough arguments.")
		fmt.Println(c.Use)
		return
	}

	// divide the commandline arguments
	image := args[0]
	cmdline := args[1:]
	wd, _ := os.Getwd()
	img_path := filepath.Join(wd, image)

	// create the initial arguments slice for the self exec
	init_args := []string{"run", image, "--name", name}
	init_args = append(init_args, "--")
	init_args = append(init_args, cmdline...)

	// check if image exists
	if _, err_stat := os.Stat(img_path); err_stat != nil {
		fmt.Println(err_stat)
		fmt.Println("Image/root filesystem not found or inaccessible at ", img_path)
		return
	}

	// debug
	fmt.Printf("Running with image '%s' and command %v as %d. :)\n", image, cmdline, os.Getpid())

	if os.Getpid() == 1 {
		// We are officially inside the container...
		cmd := exec.Command(cmdline[0], cmdline[1:]...)

		// link all the system FDs with the terminal FDs
		cmd.Stdin = os.Stdin
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout

		// set hostname to differentiate
		hname := name
		if hname == "" {
			hname = "container"
		}
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
		err_run := cmd.Run()
		if err_run != nil {
			panic(err_run)
		}
	} else if len(cmdline) != 0 {
		if os.Geteuid() == 0 && user == 0 {
			// set up the other namespaces as the host with root user (in semi-container)
			cmd := exec.Command("/proc/self/exe", init_args...)

			// link all the system FDs with the terminal FDs
			cmd.Stdin = os.Stdin
			cmd.Stderr = os.Stderr
			cmd.Stdout = os.Stdout

			// Add to the manager when a new container is opened
			if name == "" {
				newname, err_hash := utils.GenerateRandomHash(8) // generate a name based on random hash
				if err_hash != nil {
					name = "random1234"
				} else {
					name = newname
				}
			}

			// create cgroup
			cgroup_dir := filepath.Join("/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/dockman", name)
			err_cgroup := os.Mkdir(cgroup_dir, 0755)
			defer os.Remove(cgroup_dir)

			if err_cgroup != nil {
				fmt.Println("Cgroup err", err_cgroup)
				panic(err_cgroup)
			}
			cg_fd, err_fd := unix.Open(cgroup_dir, unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
			if err_fd != nil {
				panic(err_fd)
			}
			defer unix.Close(cg_fd)

			// Namespaces
			cmd.SysProcAttr = &syscall.SysProcAttr{
				Cloneflags:   unix.CLONE_NEWUTS | unix.CLONE_NEWPID | unix.CLONE_NEWNET | unix.CLONE_NEWNS,
				Unshareflags: unix.CLONE_NEWNS, // unshare the mount namespace to not show any mounts from the container. it's shared by default.
				CgroupFD:     cg_fd,            // add to container cgroup
				UseCgroupFD:  true,
			}

			// start the container runtime
			err_run := cmd.Start()
			if err_run != nil {
				panic(err_run)
			}

			defer cmd.Wait()
		} else if user == os.Geteuid() || os.Geteuid() == 0 && user != 0 {
			// set up the user namespace for container as the host user rootless
			cmd := exec.Command("/proc/self/exe", init_args...)

			// link all the system FDs with the terminal FDs
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			// Namespaces
			cmd.SysProcAttr = &syscall.SysProcAttr{
				Cloneflags: unix.CLONE_NEWUSER,
				UidMappings: []syscall.SysProcIDMap{
					{
						ContainerID: 0, HostID: user, Size: 1,
					},
				},
				GidMappings: []syscall.SysProcIDMap{
					{
						ContainerID: 0, HostID: user, Size: 1,
					},
				},
				Credential: &syscall.Credential{
					Uid: 0,
					Gid: 0,
				},
			}

			// start the container runtime
			err_run := cmd.Run()
			if err_run != nil {
				panic(err_run)
			}
		} else {
			fmt.Println("You need root to create rooted container. To create a rootless container please use the --user flag.")
		}
	}
}
