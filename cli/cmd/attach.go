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

var attach_cmd = &cobra.Command{
	Use:   "attach [flags] -- <command>",
	Short: "Run a container runtime with image and command (attaches the stdin, stdout and stderr of the command to shell)",
	Run:   attachFunc,
}

func init() {
	root_cmd.AddCommand(attach_cmd)
}

func attachFunc(cmd *cobra.Command, args []string) {
	if len(args) < 1 {
		fmt.Println("Not enough arguments.")
		fmt.Println("Usage:", cmd.Use)
	}
	// Set up a connection to the server.
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewContainerServiceClient(conn)

	r, err := c.ContainerStatus(context.Background(), &pb.ContainerIdNameRequest{ContainerIdName: args[0]})
	if err != nil {
		log.Fatalf("could not get container info: %v", err)
	}

	// attach container after checking id
	stream, err := c.AttachContainer(context.Background())
	if err != nil {
		panic(err)
	}

	// send id of container
	stream.Send(&pb.AttachContainerMessage{Payload: &pb.AttachContainerMessage_ContainerId{r.GetId()}})
	fmt.Println("Connecting to container")

	// put local terminal in raw mode, restore on exit
	if r.Pty {
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
