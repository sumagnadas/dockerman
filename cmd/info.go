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

var info_cmd = &cobra.Command{
	Use:   "info <cont_id_or_name>",
	Short: "Get info of a created container",
	RunE:  infoFunc,
}

func init() {
	root_cmd.AddCommand(info_cmd)
}

func infoFunc(cmd *cobra.Command, args []string) error {
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

	r, err := c.ContainerStatus(context.Background(), &pb.ContainerIdNameRequest{ContainerIdName: args[0]})
	if err != nil {
		fmt.Println("could not get container info:", err)
		return nil
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
	return nil
}
