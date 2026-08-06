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

var info_cmd = &cobra.Command{
	Use:   "info <cont_id_or_name>",
	Short: "Get info of a created container",
	Run:   infoFunc,
}

func init() {
	root_cmd.AddCommand(info_cmd)
}

func infoFunc(cmd *cobra.Command, args []string) {
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

	r, err := c.ContainerStatus(context.Background(), &pb.ContainerIdNameRequest{ContainerIdName: args[0]})
	if err != nil {
		log.Fatalf("could not get container info: %v", err)
	}

	root_state := "Rootless"
	if r.Rooted {
		root_state = "Rooted"
	}

	var state string
	switch r.State {
	case pb.ContainerState_RUNNING:
		state = "Running"
	case pb.ContainerState_STOPPED:
		state = "Stopped"
	case pb.ContainerState_FROZEN:
		state = "Frozen"
	}

	fmt.Println("ID:", r.Id)
	fmt.Println("Name:", r.Name)
	fmt.Println("State:", state)
	fmt.Println("Image:", r.Image)
	fmt.Println("Nprocs:", len(r.Procs))
	fmt.Println("Procs:", r.Procs)
	fmt.Println("Rooted:", root_state)
}
