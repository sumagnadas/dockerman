package cmd

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	pb "dock/service"

	"golang.org/x/term"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var run_cmd = &cobra.Command{
	Use:   "run [flags] -- <command>",
	Short: "Run a container runtime with image and command (attaches the stdin, stdout and stderr of the command to shell)",
	Run:   runFunc,
}

var user, pty bool
var name string

func init() {
	root_cmd.AddCommand(run_cmd)
	run_cmd.Flags().BoolVarP(&user, "user", "u", false, "Start an unprivileged container, mapping the current UID")
	run_cmd.Flags().BoolVarP(&pty, "tty", "t", false, "Allocate a pseudo-TTY")
	run_cmd.Flags().StringVar(&name, "name", "", "Name of the container")
}

func runFunc(cmd *cobra.Command, args []string) {
	// Set up a connection to the server.
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewContainerServiceClient(conn)
	cont_request := pb.CreateContainerRequest{Name: &name, Image: args[0], Command: args[1], Args: args[2:], Pty: pty}

	// if unprivileged container required
	if user {
		var uid int32 = int32(os.Getuid())
		cont_request.User = &uid
	}

	// Create container
	r, err := c.CreateContainer(context.Background(), &cont_request)
	if err != nil {
		log.Fatalf("could not create container: %v", err)
	}
	log.Printf("Id: %s", r.GetId())

	stream, err := c.AttachContainer(context.Background())
	if err != nil {
		panic(err)
	}

	// send id of container
	stream.Send(&pb.AttachContainerMessage{Payload: &pb.AttachContainerMessage_ContainerId{r.GetId()}})
	fmt.Println("Connecting to container")

	// put local terminal in raw mode, restore on exit
	if pty {
		oldState, _ := term.MakeRaw(int(os.Stdin.Fd()))
		defer term.Restore(int(os.Stdin.Fd()), oldState)
	}

	// stdin
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				return
			}
			stream.Send(
				&pb.AttachContainerMessage{
					Payload: &pb.AttachContainerMessage_StdinData{buf[:n]},
				},
			)
		}
	}()

	// stdout
	for {
		msg, err := stream.Recv()
		if err != nil {
			break
		}
		if data := msg.GetStdoutData(); data != nil {
			os.Stdout.Write(data)
		}
	}
}
