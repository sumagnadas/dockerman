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

var start_cmd = &cobra.Command{
	Use:   "start <cont_id_or_name>",
	Short: "Start a stopped container",
	RunE:  startFunc,
}

func init() {
	root_cmd.AddCommand(start_cmd)
}

func startFunc(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
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

	_, err = c.StartContainer(context.Background(), &pb.ContainerIdNameRequest{ContainerIdName: args[0]})
	if err != nil {
		fmt.Println("could not start container:", err)
		return nil
	}
	return nil
}
