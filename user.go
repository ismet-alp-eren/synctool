package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
)

type User struct {
	Username 		string `json:"username"`
	Token			string `json:"token"`
	ReposLocation 	string `json:"location"`
}

func (j *User) Configurate() error {
	file, err := os.Open("info.json")
	if err != nil {
		return err
	}
	log.Println("info.json is opened")
	defer file.Close()

	bValue, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	json.Unmarshal(bValue, j)
	return nil
}