package main

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"
)

type group struct {
	Name        string `json:"group_name"`
	Description string `json:"group_description"`
	GroupID     int    `json:"group_id"`
	ParentID    int    `json:"parent_id"`
}

type task struct {
	TaskID        string    `json:"task_id"`
	GroupID       int       `json:"group_id"`
	Task          string    `json:"task"`
	Completed     bool      `json:"completed"`
	CreatedDate   time.Time `json:"created_at"`
	CompletedDate time.Time `json:"completed_at"`
}

var taskGroups = readGroups()

var tasks = readTasks()

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

func readTasks() []task {
	tasksFile, err := ioutil.ReadFile("tasks.json")
	if err != nil {
		log.Fatal(err)
	}
	var newTasks []task
	err = json.Unmarshal(tasksFile, &newTasks)
	if err != nil {
		log.Fatal(err)
	}
	return newTasks
}

func groupsListHandler(w http.ResponseWriter, r *http.Request) {
	method := r.Method
	if method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
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
		newGroups = sortGroupsByName(g, 0, len(g))
	case "parents_first":
		newGroups = sortByParentsFirst(g)
	case "parent_with_children":
		newGroups = sortByParentWithChildren(g)
	default:
		newGroups = g
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

func sortGroupsByName(grs []group, s int, e int) []group {
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
		grs = sortGroupsByName(grs, s, c)
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
	grs = sortGroupsByName(grs, 0, len(grs))
	return topParents
}

func groupsHandler(w http.ResponseWriter, r *http.Request) {
	method := r.Method
	switch method {
	case "GET":
	case "POST":
	case "PUT":
	case "DELETE":

	}
}

func tasksListHandler(w http.ResponseWriter, r *http.Request) {
	method := r.Method
	if method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	l := r.URL.Query().Get("limit")
	s := r.URL.Query().Get("sort")
	t := r.URL.Query().Get("type")
	newTasks := getSortedTasks(tasks, s, l, t)
	err := json.NewEncoder(w).Encode(newTasks)
	if err != nil {
		log.Fatal(err)
	}
}

func getSortedTasks(ts []task, s string, l string, t string) []task {
	var newTasks []task
	switch s {
	case "name":
		newTasks = sortTasksByName(ts)
	case "group":
		newTasks = sortTasksByGroup(ts)
	default:
		newTasks = ts
	}
	switch t {
	case "completed":
		newTasks = getCompletedTasks(ts)
	case "working":
		newTasks = getWorkingTasks(ts)
	}
	lim, err := strconv.Atoi(l)
	if err != nil || lim < 0 {
		return ts
	}
	if lim > len(ts) {
		lim = len(ts)
	}
	newTasks = newTasks[:lim]
	return newTasks
}

func sortTasksByName(ts []task) []task {
	for i := 0; i < len(ts); i++ {
		if ts[i].Task < ts[i-1].Task {
			t := ts[i]
			g := i
			for g > 0 && t.Task < ts[g-1].Task {
				ts[g] = ts[g-1]
				g--
			}
			ts[g] = t
		}
	}
	return ts
}

func sortTasksByGroup(ts []task) []task {
	for i := 0; i < len(ts); i++ {
		if ts[i].GroupID < ts[i-1].GroupID {
			t := ts[i]
			g := i
			for g > 0 && t.GroupID < ts[g-1].GroupID {
				ts[g] = ts[g-1]
				g--
			}
			ts[g] = t
		}
	}
	return ts
}

func getCompletedTasks(ts []task) []task {
	for i := 0; i < len(ts); i++ {
		if !ts[i].Completed {
			ts = removeTask(ts, i)
		}
	}
	return ts
}

func getWorkingTasks(ts []task) []task {
	for i := 0; i < len(ts); i++ {
		if ts[i].Completed {
			ts = removeTask(ts, i)
		}
	}
	return ts
}

func removeTask(ts []task, n int) []task {
	for i := n; i < len(ts)-1; i++ {
		ts[i] = ts[i+1]
	}
	ts = ts[:len(ts)-1]
	return ts
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/groups", groupsListHandler)
	r.HandleFunc("/groups/", groupsHandler)
	r.HandleFunc("/tasks", tasksListHandler)
	r.Handle("/", r)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
