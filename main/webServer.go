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

type task struct {
}

var taskGroups = readGroups()

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
	l := r.URL.Query().Get("limit")
	s := r.URL.Query().Get("sort")
	newGroups := getSortedGroups(taskGroups, s, l)
	err := json.NewEncoder(w).Encode(newGroups)
	if err != nil {
		log.Fatal(err)
	}
}

func getSortedGroups(g []group, s string, l string) []group {
	var newGroups []group
	switch s {
	case "name":
		newGroups = sortByName(g, 0, len(g))
	case "parents_first":
		newGroups = sortByParentsFirst(g)
	case "parent_with_children":
		newGroups = sortByParentWithChildren(g)
	}
	lim, err := strconv.Atoi(l)
	if err != nil || lim < 0 {
		return g
	}
	if lim > len(g) {
		lim = len(g)
	}
	newGroups = newGroups[:lim]
	return newGroups
}

func sortByName(grs []group, s int, e int) []group {
	for i := s + 1; i < e; i++ {
		if grs[i].Name < grs[i-1].Name {
			gr := grs[i]
			g := i
			for g > 0 && gr.Name < grs[g-1].Name {
				grs[g] = grs[g-1]
				g--
			}
			grs[g] = gr
		}
	}
	return grs
}

func sortByParentsFirst(grs []group) []group {
	parentID := 0
	c := 0
	for i := 0; i < len(grs); i++ {
		s := c
		for g := c; g < len(grs); g++ {
			if parentID == grs[g].ParentID {
				grs[c], grs[g] = grs[g], grs[c]
				c++
			}
		}
		grs = sortByName(grs, s, c)
		parentID = grs[i].GroupID
	}
	return grs
}

func sortByParentWithChildren(grs []group) []group {
	var gr group
	var aGrs []group
	for k := 0; k < len(grs); k++ {
		for i := 0; i < len(grs); i++ {
			for g := 0; g < len(grs); g++ {
				if grs[i].ParentID == grs[g].GroupID {
					if contains(aGrs, grs[i]) {
						continue
					}
					gr = grs[i]
					if i < g {
						for j := i; j < g; j++ {
							grs[j] = grs[j+1]
						}
						grs[g] = gr
					} else {
						for j := i; j > g+1; j-- {
							grs[j] = grs[j-1]
						}
						grs[g+1] = gr
					}
					aGrs = append(aGrs, gr)
					i--
					break
				}
			}
		}
	}
	return grs
}

func contains(grs []group, gr group) bool {
	for i := 0; i < len(grs); i++ {
		if gr == grs[i] {
			return true
		}
	}
	return false
}

func getTopParents(grs []group) []group {
	var topParents []group
	for i := 0; i < len(grs); i++ {
		if grs[i].ParentID == 0 {
			topParents = append(topParents, grs[i])
		}
	}
	grs = sortByName(grs, 0, len(grs))
	return topParents
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/groups", groupsHandler)
	http.Handle("/", r)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
