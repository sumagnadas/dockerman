package cmd

import (
	"context"
	"errors"
	"log"

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
		log.Fatalf("did not connect to daemon: %v", err)
	}
	defer conn.Close()
	c := pb.NewContainerServiceClient(conn)

	_, err = c.StopContainer(context.Background(), &pb.ContainerIdNameRequest{ContainerIdName: args[0]})
	if err != nil {
		log.Fatalf("could not stop container: %v", err)
	}
	return nil
}
