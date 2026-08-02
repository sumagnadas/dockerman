package utils

import (
	pb "dock/service"
	"io"
)

type ContState struct {
	*pb.Container
	Stdin  io.Writer
	Stdout io.Reader
	Stderr io.Reader
}
