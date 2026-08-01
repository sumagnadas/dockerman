package cmd

import (
	"context"
	"dock/utils"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"

	pb "dock/service"

	"github.com/creack/pty"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
)

var containers = []utils.ContState{}

type ContainerServer struct {
	pb.UnimplementedContainerServiceServer
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
	init_args := []string{"run", req.Image, "--name", name}
	if req.GetUser() != 0 {
		init_args = append(init_args, "--user", strconv.Itoa(int(req.GetUser())))
		rooted = false
	}
	init_args = append(init_args, "--", req.Command)
	cmd := exec.Command("dockmanc", append(init_args, req.Args...)...)

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
	containers = append(containers, utils.ContState{Name: name, Image: req.Image, Nprocs: 1, Procs: []int{cmd.Process.Pid}, Running: true, Rooted: rooted, Stdin: stdin, Stdout: stdout, Stderr: stderr})

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
	for _, cont := range containers {
		if cont.Name == id {
			stdin = cont.Stdin
			stdout = cont.Stdout
			stderr = cont.Stderr
		}
	}
	if stdin == nil {
		fmt.Println("No such container.")
	}

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
		stream.Send(&pb.AttachContainerMessage{Payload: &pb.AttachContainerMessage_StdoutData{buf[:n]}})
	}
}

var daemon_cmd = &cobra.Command{
	Use:   "daemon",
	Short: "Launch a daemon to manage containers.",
	Run:   daemonFunc,
}

var port int

func init() {
	root_cmd.AddCommand(daemon_cmd)
	daemon_cmd.Flags().IntVarP(&port, "port", "p", 4033, "Specify an alternate port for the daemon")
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
