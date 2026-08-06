package cmd

import (
	"context"
	"fmt"

	pb "dockman/service"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var ps_cmd = &cobra.Command{
	Use:   "ps",
	Short: "Get a list of all containers",
	Run:   psFunc,
}

func init() {
	root_cmd.AddCommand(ps_cmd)
}

func psFunc(cmd *cobra.Command, args []string) {
	// Set up a connection to the server.
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Println("did not connect to daemon:", err)
		return
	}
	defer conn.Close()
	c := pb.NewContainerServiceClient(conn)

	r, err := c.ListContainers(context.Background(), &pb.EmptyMessage{})
	if err != nil {
		fmt.Println("could not list container:", err)
		return
	}
	fmt.Println("ID\t\tName\t\tImage\tNprocs\tRooted\t\tState")
	for _, cont := range r.GetConts() {
		// State strings
		root_state := "Rootless"
		if cont.Rooted {
			root_state = "Rooted\t"
		}
		var state string
		switch cont.State {
		case pb.ContainerState_RUNNING:
			state = "Running"
		case pb.ContainerState_STOPPED:
			state = "Stopped"
		case pb.ContainerState_FROZEN:
			state = "Frozen"
		}
		fmt.Printf("%s\t%s\t%s\t%d\t%s\t%s\n", cont.Id, cont.Name, cont.Image, len(cont.Procs), root_state, state)
	}
}
