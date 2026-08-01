package utils

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

/*
Generate random hash of fixed length, used for container naming
*/
func GenerateRandomHash(length int) (string, error) {
	// Allocate a byte slice to hold half the requested length
	// (since each byte produces 2 hex characters)
	bytes := make([]byte, length/2)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}

// Gets container from daemon server
func GetCont(name string) (ContState, error) {
	resp, err_get := http.Get("http://localhost:4033/get?name=" + name)
	if err_get != nil {
		fmt.Println(err_get)
		return ContState{}, err_get
	}
	if resp.StatusCode == 500 {
		fmt.Println("Container with name", name, "does not exist.")
		return ContState{}, errors.New("Container does not exist.")
	}

	body, err_read := io.ReadAll(resp.Body)
	if err_read != nil {
		fmt.Println("Couldn't read body.", err_read)
		return ContState{}, err_read
	}
	var cont ContState
	err_json := json.Unmarshal(body, &cont)
	if err_json != nil {
		fmt.Println("Not exactly json?", err_json)
		return ContState{}, err_json
	}
	return cont, nil
}

// Update container info in daemon
func UpdateCont(name string, cont ContState) error {
	upd_cont, _ := json.Marshal(cont)
	_, err_upd := http.Post("http://localhost:4033/update", "application/json", bytes.NewBuffer(upd_cont))
	if err_upd != nil {
		fmt.Println("Couldn't update container.", err_upd)
		return err_upd
	}
	return nil
}
