package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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

	// remove cgroup
	cgroup_dir := filepath.Join("/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/app.slice/dockerman", name)
	os.Remove(cgroup_dir)
}
