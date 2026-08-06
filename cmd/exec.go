package cmd

import (
	"context"
	"errors"
	"log"
	"os"

	"golang.org/x/term"

	"github.com/spf13/cobra"

	pb "dockman/service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var exec_cmd = &cobra.Command{
	Use:   "exec [flags] <container> <command>",
	Short: "Exec into a container with a command",
	RunE:  execFunc,
}

var interactive_exec, pty_exec bool

func init() {
	root_cmd.AddCommand(exec_cmd)
	exec_cmd.Flags().BoolVarP(&interactive_exec, "interactive", "i", false, "Keep STDIN open if not attached")
	exec_cmd.Flags().BoolVarP(&pty_exec, "tty", "t", false, "Allocate a pseudo-TTY")
}

func execFunc(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return errors.New("Not enough arguments.")
	}
	// Set up a connection to the server.
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect to daemon: %v", err)
	}
	defer conn.Close()
	c := pb.NewContainerServiceClient(conn)

	stream, err_exec := c.Exec(context.Background())
	if err_exec != nil {
		log.Fatalf("could not exec: %v", err)
	}

	// send container id and process to make
	stream.Send(&pb.ExecContainerMessage{Payload: &pb.ExecContainerMessage_Proc{&pb.ExecProcess{ContainerId: args[0], Cmdline: args[1:], Interactive: interactive_exec, Pty: pty_exec}}})
	if interactive_exec {
		if pty_exec {
			// send raw bytes from terminal without catching anything.
			oldState, _ := term.MakeRaw(int(os.Stdin.Fd()))
			defer term.Restore(int(os.Stdin.Fd()), oldState)
		}

		// stdin sync
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
					&pb.ExecContainerMessage{
						Payload: &pb.ExecContainerMessage_StdinData{buf[:n]},
					},
				)
			}
		}()

		// stdout sync
		for {
			msg, err := stream.Recv()
			if err != nil || err_in != nil {
				log.Fatalf("Container stopped or connection with daemon broke.")
			}
			if data := msg.GetStdoutData(); data != nil {
				os.Stdout.Write(data)
			}
		}
	}
	return nil
}
