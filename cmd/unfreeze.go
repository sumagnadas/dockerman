package cmd

import (
	"context"
	pb "dockman/service"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var unfreeze_cmd = &cobra.Command{
	Use:   "unfreeze <cont_id_or_name>",
	Short: "Unfreeze a container",
	RunE:  unfreezeFunc,
}

func init() {
	root_cmd.AddCommand(unfreeze_cmd)
}

func unfreezeFunc(cmd *cobra.Command, args []string) error {
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

	_, err = c.UnfreezeContainer(context.Background(), &pb.ContainerIdNameRequest{ContainerIdName: args[0]})
	if err != nil {
		fmt.Println("could not unfreeze container:", err)
		return nil
	}
	return nil
}
