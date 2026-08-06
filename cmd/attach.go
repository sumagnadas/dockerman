package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	pb "dockman/service"

	"golang.org/x/term"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var attach_cmd = &cobra.Command{
	Use:   "attach <container> <command>",
	Short: "Attaches the stdin, stdout and stderr of the command to the container)",
	RunE:  attachFunc,
}

func init() {
	root_cmd.AddCommand(attach_cmd)
}

func attachFunc(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return errors.New("Not enough arguments.")
	}
	// Set up a connection to the server.
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Println("did not connect to daemon:", err)
		return nil
	}
	defer conn.Close()
	c := pb.NewContainerServiceClient(conn)

	r, err := c.ContainerStatus(context.Background(), &pb.ContainerIdNameRequest{ContainerIdName: args[0]})
	if err != nil {
		fmt.Println("could not get container info:", err)
		return nil
	}

	// attach container after checking id
	stream, err := c.AttachContainer(context.Background())
	if err != nil {
		fmt.Println("did not attach properly:", err)
		return nil
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
	var err_in error
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				err_in = err
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
		if err != nil || err_in != nil {
			fmt.Println("Container stopped or connection with daemon broke.")
			return nil
		}
		if data := msg.GetStdoutData(); data != nil {
			os.Stdout.Write(data)
		}
	}
}
