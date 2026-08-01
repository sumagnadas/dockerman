package utils

import (
	"io"
)

type ContState struct {
	Id      string  `json:"id"`
	Name    string  `json:"name"`
	Image   string  `json:"image"`
	Nprocs  int32   `json:"nprocs"`  // No. of main/starting process(es)
	Procs   []int32 `json:"procs"`   // PID of the main/starting process(es)
	Running bool    `json:"running"` // Status of the container (Running/stopped)
	Rooted  bool    `json:"rooted"`
	Stdin   io.Writer
	Stdout  io.Reader
	Stderr  io.Reader
}
