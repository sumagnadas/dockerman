package utils

type ContState struct {
	Name   string `json:"name"`
	Image  string `json:"image"`
	Nprocs int    `json:"nprocs"` // No. of main/starting process(es)
	Procs  []int  `json:"procs"`  // PID of the main/starting process(es)
}
