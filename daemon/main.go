package main

import (
	"bufio"
	"context"
	"dockman/utils"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"

	pb "dockman/service"

	"github.com/creack/pty"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
)

var containers = []*utils.ContState{}

type ContainerServer struct {
	pb.UnimplementedContainerServiceServer
}

func readLine(r io.Reader) (string, error) {
	scanner := bufio.NewScanner(r)

	var line string
	if scanner.Scan() {
		line = scanner.Text()
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}
	return line, nil
}
func removePid(container *utils.ContState, pid int) {
	for i, cont_pid := range container.Procs {
		if cont_pid == int32(pid) {
			container.Procs = append(container.Procs[:i], container.Procs[i+1:]...)
			break
		}
	}

	// if all processes exit, mark the container as stopped instead of removing it.
	if len(container.Procs) == 0 {
		for _, cont_v := range containers {
			if cont_v == container {
				cont_v.State = pb.ContainerState_STOPPED
				break
			}
		}
	}
}

func (s *ContainerServer) CreateContainer(ctx context.Context, req *pb.CreateContainerRequest) (*pb.CreateContainerResponse, error) {
	var id, name string
	rooted := true

	newid, err_hash := utils.GenerateRandomHash(8) // generate a id based on random hash
	if err_hash != nil {
		id = "random1234"
	} else {
		id = newid
	}
	name = req.GetName()
	if name == "" {
		name = id
	}
	init_args := []string{req.Image, "--name", name}

	// add the uid to the container config
	user := 0
	if req.GetUser() != 0 {
		user = int(req.GetUser())
		init_args = append(init_args, "--user", strconv.Itoa(user))
		rooted = false
	}
	init_args = append(init_args, req.Command)
	cmd := exec.Command("dockmanc", append(init_args, req.Args...)...)
	cmd.Env = os.Environ()

	var stdin io.Writer
	var stdout, stderr io.Reader

	// start command with or without pty
	if req.GetPty() {
		f, err := pty.Start(cmd)
		if err != nil {
			return &pb.CreateContainerResponse{Id: "-1"}, err
		}
		stdin = f
		stdout = f
		stderr = f
	} else {
		// assign the pipes before cmd is started
		stdin_pipe, err_stdin := cmd.StdinPipe()
		stdout_pipe, err_stdout := cmd.StdoutPipe()
		stderr_pipe, err_stderr := cmd.StderrPipe()

		err_start := cmd.Start()
		if err_start != nil {
			return &pb.CreateContainerResponse{Id: "-1"}, err_start
		}
		if err_stdin != nil || err_stdout != nil || err_stderr != nil {
			cmd.Process.Signal(unix.SIGTERM)
			fmt.Println(err_stdin)
			fmt.Println(err_stdout)
			fmt.Println(err_stderr)
			return &pb.CreateContainerResponse{Id: "-1"}, errors.New("Can't connect pipe.")
		}

		stdin = stdin_pipe
		stdout = stdout_pipe
		stderr = stderr_pipe
	}
	if cmd.ProcessState.ExitCode() == 1 {
		return &pb.CreateContainerResponse{Id: "-1"}, errors.New("something happened.")
	}

	// first line of the stdout, if it has not crashed is always the PID of the enclosed command.
	// (just an arbitrary and a very bad choice of architecture)
	pid, err_pid := readLine(stdout)
	if err_pid != nil {
		return &pb.CreateContainerResponse{Id: "-1"}, err_pid
	}
	pid_int, _ := strconv.Atoi(pid)
	cont := utils.ContState{Container: &pb.Container{Id: id, Name: name, Image: req.Image, Procs: []int32{int32(pid_int)}, State: pb.ContainerState_RUNNING, Rooted: rooted, Pty: req.GetPty(), User: int32(user), Cmd: req.Command, Args: req.Args}, Stdin: stdin, Stdout: stdout, Stderr: stderr}
	containers = append(containers, &cont)

	// cleanup function for the container process
	go func() {
		cmd.Wait()
		removePid(&cont, pid_int)
	}()
	return &pb.CreateContainerResponse{Id: name}, nil
}

func (s *ContainerServer) AttachContainer(stream pb.ContainerService_AttachContainerServer) (err error) {
	msg, err_stream := stream.Recv()
	if err_stream != nil {
		return err_stream
	}
	id := msg.GetContainerId()
	if id == "" {
		return errors.New("No container id sent.")
	}

	var stdin io.Writer
	var stdout, stderr io.Reader
	var cont_req *utils.ContState
	for _, cont := range containers {
		if cont.Name == id || cont.Id == id {
			stdin = cont.Stdin
			stdout = cont.Stdout
			stderr = cont.Stderr
			cont_req = cont
		}
	}
	if stdin == nil {
		fmt.Println("No such container.")
		return errors.New("No such container.")
	}

	output := io.MultiReader(stdout, stderr)

	// input sync
	go func() {
		for {
			if cont_req.State == pb.ContainerState_STOPPED {
				return
			}
			msg, err := stream.Recv()
			if err != nil {
				return
			}
			stdin.Write(msg.GetStdinData())
		}
	}()

	// output sync
	buf := make([]byte, 4096)
	for {
		if cont_req.State == pb.ContainerState_STOPPED {
			return errors.New("Container is stopped. Please start the container to exec into it.")
		}
		n, err_read := output.Read(buf)
		if err_read != nil {
			return err_read
		}
		stream.Send(&pb.AttachContainerMessage{Payload: &pb.AttachContainerMessage_StdoutData{buf[:n]}})
	}
}

func (s *ContainerServer) ContainerStatus(ctx context.Context, req *pb.ContainerIdNameRequest) (*pb.Container, error) {
	var cont_stat *pb.Container

	// Check both ID and name, whatever comes first
	for _, cont := range containers {
		if cont.Name == req.GetContainerIdName() || cont.Id == req.GetContainerIdName() {
			cont_stat = cont.Container
		}
	}

	if cont_stat != nil {
		return cont_stat, nil
	} else {
		return &pb.Container{}, errors.New("Couldn't find container.")
	}
}

func (s *ContainerServer) ListContainers(ctx context.Context, req *pb.EmptyMessage) (*pb.ListContainersResponse, error) {
	var cont_list []*pb.Container
	for _, cont := range containers {
		cont_list = append(cont_list, cont.Container)
	}
	return &pb.ListContainersResponse{Conts: cont_list}, nil
}

func (s *ContainerServer) Exec(stream pb.ContainerService_ExecServer) (err error) {
	msg, err_stream := stream.Recv()
	if err_stream != nil {
		return err_stream
	}
	proc := msg.GetProc()
	if proc == nil {
		return errors.New("No container id sent.")
	}

	var cont_req *utils.ContState
	for _, cont := range containers {
		if cont.Id == proc.ContainerId || cont.Name == proc.ContainerId {
			cont_req = cont
			break
		}
	}
	if cont_req == nil {
		return errors.New("No container with this ID or name.")
	}
	if cont_req.State == pb.ContainerState_STOPPED {
		return errors.New("Container is stopped. Please start the container to exec into it.")
	}

	target_pid := cont_req.Procs[0]
	ns_args := []string{"-t", strconv.Itoa(int(target_pid)), "--all"}
	ns_args = append(ns_args, proc.Cmdline...)

	// enter container cgroup
	cgroup_dir := filepath.Join("/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/dockman", cont_req.Name)
	cg_fd, err_fd := unix.Open(cgroup_dir, unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err_fd != nil {
		return err_fd
	}
	defer unix.Close(cg_fd)

	nscmd := exec.Command("nsenter", ns_args...)
	nscmd.SysProcAttr = &syscall.SysProcAttr{
		CgroupFD:    cg_fd, // add to container cgroup
		UseCgroupFD: true,
	}
	var stdin io.Writer
	var stdout, stderr io.Reader

	if proc.GetPty() {
		f, err := pty.Start(nscmd)
		if err != nil {
			return err
		}
		stdin = f
		stdout = f
		stderr = f
	} else {
		stdin_pipe, err_stdin := nscmd.StdinPipe()
		stdout_pipe, err_stdout := nscmd.StdoutPipe()
		stderr_pipe, err_stderr := nscmd.StderrPipe()

		err_run := nscmd.Start()
		if err_run != nil {
			panic(err_run)
		}
		if err_stdin != nil || err_stdout != nil || err_stderr != nil {
			nscmd.Process.Signal(unix.SIGTERM)
			fmt.Println(err_stdin)
			fmt.Println(err_stdout)
			fmt.Println(err_stderr)
			return errors.New("Can't connect pipe.")
		}

		stdin = stdin_pipe
		stdout = stdout_pipe
		stderr = stderr_pipe
	}
	cont_req.Procs = append(cont_req.Procs, int32(nscmd.Process.Pid))
	go func() {
		nscmd.Wait()
		removePid(cont_req, nscmd.Process.Pid)
	}()

	if proc.GetInteractive() {
		output := io.MultiReader(stdout, stderr)
		// input sync
		go func() {
			for {
				msg, err := stream.Recv()
				if err != nil {
					return
				}
				stdin.Write(msg.GetStdinData())
			}
		}()

		// output sync
		buf := make([]byte, 4096)
		for {
			n, err_read := output.Read(buf)
			if err_read != nil {
				return err_read
			}
			stream.Send(&pb.ExecContainerMessage{Payload: &pb.ExecContainerMessage_StdoutData{buf[:n]}})
		}
	}
	return nil
}

func (s *ContainerServer) RemoveContainer(ctx context.Context, req *pb.ContainerIdNameRequest) (*pb.EmptyMessage, error) {
	id := -1
	for ind, cont := range containers {
		if cont.Id == req.ContainerIdName || cont.Name == req.ContainerIdName {
			id = ind
			break
		}
	}
	if id == -1 {
		return &pb.EmptyMessage{}, errors.New("No container with this ID or name.")
	} else {
		if containers[id].State == pb.ContainerState_RUNNING {
			return &pb.EmptyMessage{}, errors.New("Container is running. Stop it to remove it.")
		}
		// Remove the container from the daemon memory
		containers = append(containers[:id], containers[id+1:]...)
		return &pb.EmptyMessage{}, nil
	}

}
func (s *ContainerServer) StopContainer(ctx context.Context, req *pb.ContainerIdNameRequest) (*pb.EmptyMessage, error) {
	var cont_req *utils.ContState
	for _, cont := range containers {
		if cont.Id == req.ContainerIdName || cont.Name == req.ContainerIdName {
			cont_req = cont
			break
		}
	}
	if cont_req == nil {
		return &pb.EmptyMessage{}, errors.New("No container with this ID or name.")
	}

	// kill all the processes of the container to stop it.
	cgroup_kill_file := filepath.Join("/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/dockman", cont_req.Name, "cgroup.kill")
	os.WriteFile(cgroup_kill_file, []byte{byte('1')}, 0755)

	cont_req.State = pb.ContainerState_STOPPED

	return &pb.EmptyMessage{}, nil
}

func (s *ContainerServer) StartContainer(ctx context.Context, req *pb.ContainerIdNameRequest) (*pb.EmptyMessage, error) {
	var cont_req *utils.ContState
	for _, cont := range containers {
		if cont.Id == req.ContainerIdName || cont.Name == req.ContainerIdName {
			cont_req = cont
			break
		}
	}
	if cont_req == nil {
		return &pb.EmptyMessage{}, errors.New("No container with this ID or name.")
	}

	if cont_req.State == pb.ContainerState_RUNNING {
		return &pb.EmptyMessage{}, errors.New("Container is already running.")
	}

	init_args := []string{cont_req.Image, "--name", cont_req.Name}
	if !cont_req.Rooted {
		init_args = append(init_args, "--user", strconv.Itoa(int(cont_req.User)))
	}
	init_args = append(init_args, cont_req.Cmd)
	cmd := exec.Command("dockmanc", append(init_args, cont_req.Args...)...)

	var stdin io.Writer
	var stdout, stderr io.Reader

	// start command with or without pty
	if cont_req.GetPty() {
		f, err := pty.Start(cmd)
		if err != nil {
			fmt.Println(err)
			return &pb.EmptyMessage{}, err
		}
		fmt.Println(f.Name())
		stdin = f
		stdout = f
		stderr = f
	} else {
		// assign the pipes before cmd is started
		stdin_pipe, err_stdin := cmd.StdinPipe()
		stdout_pipe, err_stdout := cmd.StdoutPipe()
		stderr_pipe, err_stderr := cmd.StderrPipe()

		err_start := cmd.Start()
		if err_start != nil {
			return &pb.EmptyMessage{}, err_start
		}
		if err_stdin != nil || err_stdout != nil || err_stderr != nil {
			cmd.Process.Signal(unix.SIGTERM)
			fmt.Println(err_stdin)
			fmt.Println(err_stdout)
			fmt.Println(err_stderr)
			return &pb.EmptyMessage{}, errors.New("Can't connect pipe.")
		}
		stdin = stdin_pipe
		stdout = stdout_pipe
		stderr = stderr_pipe
	}
	if cmd.ProcessState.ExitCode() == 1 {
		return &pb.EmptyMessage{}, errors.New("something happened.")
	}
	// if its running, then its running on hopes and dreams
	pid, err_pid := readLine(stdout)
	if err_pid != nil {
		return &pb.EmptyMessage{}, err_pid
	}
	pid_int, _ := strconv.Atoi(pid)
	cont_req.Procs = append(cont_req.Procs, int32(pid_int))
	cont_req.State = pb.ContainerState_RUNNING

	cont_req.Stdin = stdin
	cont_req.Stdout = stdout
	cont_req.Stderr = stderr

	// cleanup function
	go func() {
		cmd.Wait()
		removePid(cont_req, pid_int)
	}()
	return &pb.EmptyMessage{}, nil
}

func (s *ContainerServer) FreezeContainer(ctx context.Context, req *pb.ContainerIdNameRequest) (*pb.EmptyMessage, error) {
	var cont_req *utils.ContState
	for _, cont := range containers {
		if cont.Id == req.ContainerIdName || cont.Name == req.ContainerIdName {
			cont_req = cont
			break
		}
	}
	if cont_req == nil {
		return &pb.EmptyMessage{}, errors.New("No container with this ID or name.")
	}
	if cont_req.State == pb.ContainerState_STOPPED {
		return &pb.EmptyMessage{}, errors.New("Stopped container cannot be frozen.")
	}

	// freeze all the processes of the container to stop it.
	cgroup_freeze_file := filepath.Join("/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/dockman", cont_req.Name, "cgroup.freeze")
	os.WriteFile(cgroup_freeze_file, []byte{byte('1')}, 0755)

	cont_req.State = pb.ContainerState_FROZEN

	return &pb.EmptyMessage{}, nil
}
func (s *ContainerServer) UnfreezeContainer(ctx context.Context, req *pb.ContainerIdNameRequest) (*pb.EmptyMessage, error) {
	var cont_req *utils.ContState
	for _, cont := range containers {
		if cont.Id == req.ContainerIdName || cont.Name == req.ContainerIdName {
			cont_req = cont
			break
		}
	}
	if cont_req == nil {
		return &pb.EmptyMessage{}, errors.New("No container with this ID or name.")
	}
	if cont_req.State == pb.ContainerState_RUNNING || cont_req.State == pb.ContainerState_STOPPED {
		return &pb.EmptyMessage{}, errors.New("Container is not frozen.")
	}

	// unfreeze all the processes of the container to stop it.
	cgroup_freeze_file := filepath.Join("/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/dockman", cont_req.Name, "cgroup.freeze")
	os.WriteFile(cgroup_freeze_file, []byte{byte('0')}, 0755)

	cont_req.State = pb.ContainerState_RUNNING

	return &pb.EmptyMessage{}, nil
}

var root_cmd = &cobra.Command{
	Use:   "dockmand",
	Short: "Minimal container lifecycle and state management daemon",
	Run:   daemonFunc,
}

var port int

func init() {
	root_cmd.Flags().IntVarP(&port, "port", "p", 4033, "Specify an alternate port for the daemon")
}

func daemonFunc(cmd *cobra.Command, args []string) {
	// check privileges
	if os.Getuid() != 0 {
		fmt.Println("Please run the daemon as root user.")
		return
	}

	// create root cgroup if not present
	root_cgroup := "/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/dockman"
	err_cgroupdir := os.MkdirAll(root_cgroup, 0755)
	if err_cgroupdir != nil {
		fmt.Println("Couldn't set up root cgroup dir", err_cgroupdir)
		return
	}

	// change to user
	os.Chown(root_cgroup, 1000, 1000)

	// create gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterContainerServiceServer(s, &ContainerServer{})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func main() {
	if err := root_cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
