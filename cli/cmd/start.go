package cmd

import (
	"context"
	"fmt"
	"log"

	pb "dock/service"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var start_cmd = &cobra.Command{
	Use:   "start <cont_id_or_name>",
	Short: "Start a container",
	Run:   startFunc,
}

func init() {
	root_cmd.AddCommand(start_cmd)
}

func startFunc(cmd *cobra.Command, args []string) {
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

	_, err = c.StartContainer(context.Background(), &pb.ContainerIdNameRequest{ContainerIdName: args[0]})
	if err != nil {
		fmt.Println(err)
	}
}
