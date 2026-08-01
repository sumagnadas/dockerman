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

	r, err := c.ContainerStatus(context.Background(), &pb.ContainerStatusRequest{ContainerIdName: args[0]})
	if err != nil {
		log.Fatalf("could not get container info: %v", err)
	}

	root_state := "Rootless"
	if r.Cont.Rooted {
		root_state = "Rooted"
	}

	fmt.Println("ID:", r.Cont.Id)
	fmt.Println("Name:", r.Cont.Name)
	fmt.Println("Status:", r.State)
	fmt.Println("Image:", r.Cont.Image)
	fmt.Println("Nprocs:", r.Cont.Nprocs)
	fmt.Println("Procs:", r.Cont.Procs)
	fmt.Println("Rooted:", root_state)
}
