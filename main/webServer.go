package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
)

type group struct {
	Name        string `json:"group_name"`
	Description string `json:"group_description"`
	GroupID     int    `json:"group_id"`
	ParentID    int    `json:"parent_id"`
}

type task struct {
}

var groups = readGroups()

func readGroups() []group {
	groupsFile, err := ioutil.ReadFile("groups.json")
	if err != nil {
		log.Fatal(err)
	}
	var groups []group
	err = json.Unmarshal(groupsFile, &groups)
	if err != nil {
		log.Fatal(err)
	}
	return groups
}

func groupsHandler(w http.ResponseWriter, r *http.Request) {
	err := json.NewEncoder(w).Encode(groups)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	http.HandleFunc("/groups", groupsHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
