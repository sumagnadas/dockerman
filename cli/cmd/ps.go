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

var ps_cmd = &cobra.Command{
	Use:   "ps",
	Short: "Get a list of running containers",
	Run:   psFunc,
}

func init() {
	root_cmd.AddCommand(ps_cmd)
}

func psFunc(cmd *cobra.Command, args []string) {

	// Set up a connection to the server.
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewContainerServiceClient(conn)

	r, err := c.ListContainers(context.Background(), &pb.EmptyMessage{})
	if err != nil {
		log.Fatalf("could not list container: %v", err)
	}
	fmt.Println("ID\t\tName\t\tImage\tNprocs\tRooted\t\tState")
	for _, cont := range r.GetConts() {
		// State strings
		root_state := "Rootless"
		if cont.Cont.Rooted {
			root_state = "Rooted\t"
		}
		fmt.Printf("%s\t%s\t%s\t%d\t%s\t%s\n", cont.Cont.Id, cont.Cont.Name, cont.Cont.Image, cont.Cont.Nprocs, root_state, cont.State)
	}
}
