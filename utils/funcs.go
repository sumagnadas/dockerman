package utils

import (
	"crypto/rand"
	"encoding/hex"

	"fmt"
	"net/http"
	"os/exec"
	"strconv"
)

func GenerateRandomHash(length int) (string, error) {
	// Allocate a byte slice to hold half the requested length
	// (since each byte produces 2 hex characters)
	bytes := make([]byte, length/2)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}

func WaitAndRemove(cmd *exec.Cmd, name string, pid int) {
	cmd.Wait()
	_, err := http.Get("http://localhost:4033/remove?name=" + name + "&pid=" + strconv.Itoa(pid))
	if err != nil {
		fmt.Println("Get failed: ", err)
	}
}
