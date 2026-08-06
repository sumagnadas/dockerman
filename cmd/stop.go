package cmd

import (
	"context"
	"errors"
	"fmt"

	pb "dockman/service"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var stop_cmd = &cobra.Command{
	Use:   "stop <cont_id_or_name>",
	Short: "Stop a container",
	RunE:  stopFunc,
}

func init() {
	root_cmd.AddCommand(stop_cmd)
}

func stopFunc(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return errors.New("Not enough arguments")
	}
	// Set up a connection to the server.
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Println("did not connect to daemon: ", err)
		return nil
	}
	defer conn.Close()
	c := pb.NewContainerServiceClient(conn)

	_, err = c.StopContainer(context.Background(), &pb.ContainerIdNameRequest{ContainerIdName: args[0]})
	if err != nil {
		fmt.Println("could not stop container: ", err)
		return nil
	}
	return nil
}
