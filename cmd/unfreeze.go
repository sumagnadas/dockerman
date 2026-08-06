package cmd

import (
	"context"
	"fmt"
	"log"

	pb "dockman/service"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var unfreeze_cmd = &cobra.Command{
	Use:   "unfreeze <cont_id_or_name>",
	Short: "Unfreeze a container",
	Run:   unfreezeFunc,
}

func init() {
	root_cmd.AddCommand(unfreeze_cmd)
}

func unfreezeFunc(cmd *cobra.Command, args []string) {
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

	_, err = c.UnfreezeContainer(context.Background(), &pb.ContainerIdNameRequest{ContainerIdName: args[0]})
	if err != nil {
		fmt.Println(err)
	}
}
