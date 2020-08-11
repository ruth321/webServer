package main

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
)

type group struct {
	Name        string `json:"group_name"`
	Description string `json:"group_description"`
	GroupID     int    `json:"group_id"`
	ParentID    int    `json:"parent_id"`
}

type groups []group

type task struct {
}

var taskGroups = readGroups()

func readGroups() groups {
	groupsFile, err := ioutil.ReadFile("groups.json")
	if err != nil {
		log.Fatal(err)
	}
	var groups groups
	err = json.Unmarshal(groupsFile, &groups)
	if err != nil {
		log.Fatal(err)
	}
	return groups
}

func groupsHandler(w http.ResponseWriter, r *http.Request) {
	l := r.URL.Query().Get("limit")
	s := r.URL.Query().Get("sort")
	newGroups := getSortedGroups(taskGroups, s, l)
	err := json.NewEncoder(w).Encode(newGroups)
	if err != nil {
		log.Fatal(err)
	}
}

func getSortedGroups(g groups, s string, l string) groups {
	switch s {
	case "name":
		g.sortByName(0, len(g))
	case "parents_first":
		g.sortByParentsFirst()
	case "parent_with_children":
		g.sortByParentWithChildren()
	}
	lim, err := strconv.Atoi(l)
	if err != nil || lim < 0 {
		return g
	}
	if lim > len(g) {
		lim = len(g)
	}
	g = g[:lim]
	return g
}

func (grs *groups) sortByName(s int, e int) {
	for i := s + 1; i < e; i++ {
		if (*grs)[i].Name < (*grs)[i-1].Name {
			gr := (*grs)[i]
			g := i
			for g > 0 && gr.Name < (*grs)[g-1].Name {
				(*grs)[g] = (*grs)[g-1]
				g--
			}
			(*grs)[g] = gr
		}
	}
}

func (grs *groups) sortByParentsFirst() {
	parentID := 0
	c := 0
	for i := 0; i < len(*grs); i++ {
		s := c
		for g := c; g < len(*grs); g++ {
			if parentID == (*grs)[g].ParentID {
				(*grs)[c], (*grs)[g] = (*grs)[g], (*grs)[c]
				c++
			}
		}
		(*grs).sortByName(s, c)
		parentID = (*grs)[i].GroupID
	}
}

func (grs *groups) sortByParentWithChildren() {

}

func getTopParents(grs groups) groups {
	var topParents groups
	for i := 0; i < len(grs); i++ {
		if grs[i].ParentID == 0 {
			topParents = append(topParents, grs[i])
		}
	}
	grs.sortByName(0, len(grs))
	return topParents
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/groups/", groupsHandler)
	http.Handle("/", r)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
