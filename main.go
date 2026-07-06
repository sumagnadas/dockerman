package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"golang.org/x/sys/unix"
)

func main() {
	switch os.Args[1] {
	case "run":
		run()
	default:
		panic("Bad command")
	}
}

// docker         run image <cmd>
// go run main.go run image <cmd>
func run() {
	if len(os.Args) < 4 {
		fmt.Println("Not enough arguments.")
		return
	}

	// divide the commandline arguments
	image := os.Args[2]
	cmdline := os.Args[3:]

	wd, _ := os.Getwd()

	// check if image exists
	if _, err := os.Stat(wd + "/" + image); err != nil {
		fmt.Println("Image/root filesystem not found or inaccessible at ", wd+"/"+image)
		return
	}

	// debug
	fmt.Printf("Running with image '%s' and command %v as %d. :)\n", os.Args[2], os.Args[3:], os.Getpid())

	if os.Getpid() == 1 {
		// We are officially inside the container...
		cmd := exec.Command(cmdline[0], cmdline[1:]...)

		// link all the system FDs with the terminal FDs
		cmd.Stdin = os.Stdin
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout

		// set hostname to differentiate
		unix.Sethostname([]byte("container"))

		// set root and mount proc
		fmt.Println("Changing root to ", wd+"/"+image)
		unix.Chroot(wd + "/" + image)
		unix.Chdir("/")
		unix.Mount("proc", "proc", "proc", 0, "")

		// for now, make sure the filesystem is unmounted before exiting the container
		defer unix.Unmount("/proc", unix.MNT_DETACH)

		// run it
		errRun := cmd.Run()
		if errRun != nil {
			panic(errRun)
		}
	} else if len(cmdline) != 0 {
		// set up the container namespaces as the host
		cmd := exec.Command("/proc/self/exe", append([]string{"run"}, os.Args[2:]...)...)

		// link all the system FDs with the terminal FDs
		cmd.Stdin = os.Stdin
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout

		// Namespaces
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Cloneflags: unix.CLONE_NEWUTS | unix.CLONE_NEWPID | unix.CLONE_NEWNET | unix.CLONE_NEWNS,
		}

		// start the container runtime
		errRun := cmd.Run()
		if errRun != nil {
			panic(errRun)
		}
		fmt.Println("Container exited...")
	}
}
