package cmd

import (
	"context"
	"fmt"
	"log"
	"os"

	"golang.org/x/term"

	"github.com/spf13/cobra"

	pb "dock/service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var exec_cmd = &cobra.Command{
	Use:   "exec <container> [flags] -- <command>",
	Short: "Run a container runtime with image and command (execes the stdin, stdout and stderr of the command to shell)",
	Run:   execFunc,
}

var interactive_exec, pty_exec bool

func init() {
	root_cmd.AddCommand(exec_cmd)
	exec_cmd.Flags().BoolVarP(&interactive_exec, "interactive", "i", false, "Keep STDIN open if not attached")
	exec_cmd.Flags().BoolVarP(&pty_exec, "tty", "t", false, "Allocate a pseudo-TTY")
}

func execFunc(cmd *cobra.Command, args []string) {
	if len(args) < 2 {
		fmt.Println("Not enough arguments.")
		fmt.Println("Usage:", cmd.Use)
	}
	// Set up a connection to the server.
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("could not connect to daemon: %v", err)
	}
	defer conn.Close()
	c := pb.NewContainerServiceClient(conn)

	stream, err_exec := c.Exec(context.Background())
	if err_exec != nil {
		log.Fatalf("could not exec: %v", err)
	}

	stream.Send(&pb.ExecContainerMessage{Payload: &pb.ExecContainerMessage_Proc{&pb.ExecProcess{ContainerId: args[0], Cmdline: args[1:], Interactive: interactive_exec, Pty: pty_exec}}})
	if interactive_exec {
		if pty_exec {
			oldState, _ := term.MakeRaw(int(os.Stdin.Fd()))
			defer term.Restore(int(os.Stdin.Fd()), oldState)
		}
		go func() {
			buf := make([]byte, 4096)
			for {
				n, err := os.Stdin.Read(buf)
				if err != nil {
					return
				}
				stream.Send(
					&pb.ExecContainerMessage{
						Payload: &pb.ExecContainerMessage_StdinData{buf[:n]},
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

}
