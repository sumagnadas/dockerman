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

var run_cmd = &cobra.Command{
	Use:   "run [flags] <image> <command>",
	Short: "Create a container runtime with specified image and command",
	RunE:  runFunc,
}

var user, pty_run, interactive_run bool
var name string

func init() {
	root_cmd.AddCommand(run_cmd)
	run_cmd.Flags().BoolVarP(&user, "user", "u", false, "Start an unprivileged container, mapping the current UID")
	run_cmd.Flags().BoolVarP(&pty_run, "tty", "t", false, "Allocate a pseudo-TTY")
	run_cmd.Flags().BoolVarP(&interactive_run, "interactive", "i", false, "Keep STDIN open if not attached")
	run_cmd.Flags().StringVar(&name, "name", "", "Name of the container")
}

func runFunc(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return errors.New("Not enough arguments")
	}
	// Set up a connection to the server.
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Println("did not connect to daemon:", err)
		return nil
	}
	defer conn.Close()
	c := pb.NewContainerServiceClient(conn)
	cont_request := pb.CreateContainerRequest{Name: &name, Image: args[0], Command: args[1], Args: args[2:], Pty: pty_run}

	// if unprivileged container required
	if user {
		var uid int32 = int32(os.Getuid())
		cont_request.User = &uid
	}

	// Create container
	r, err := c.CreateContainer(context.Background(), &cont_request)
	if err != nil {
		fmt.Println("could not create container:", err)
		return nil
	}

	if interactive_run {

		stream, err := c.AttachContainer(context.Background())
		if err != nil {
			fmt.Println("could not attach to container:", err)
			return nil
		}

		// send id of container
		stream.Send(&pb.AttachContainerMessage{Payload: &pb.AttachContainerMessage_ContainerId{r.GetId()}})
		fmt.Println("Connecting to container")

		// put local terminal in raw mode, restore on exit
		if pty_run {
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
	return nil
}
