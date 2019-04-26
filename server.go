package main

import (
	"encoding/json"
	"log"
	"os"
)

func main() {
	gynFile, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	params := Params{}
	err = json.NewDecoder(gynFile).Decode(&params)
	if err != nil {
		log.Fatalf("Error parsing gyndns.json: %v", err)
	}

	New(&params).Run()
}
