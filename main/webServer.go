package main

import (
	"encoding/json"
	"errors"
	"github.com/gorilla/mux"
	"html/template"
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
	TaskID        string `json:"task_id"`
	GroupID       int    `json:"group_id"`
	Task          string `json:"task"`
	Completed     bool   `json:"completed"`
	CreatedDate   string `json:"created_at"`
	CompletedDate string `json:"completed_at"`
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
		newGroups = sortByParentWithChildren(g, 0)
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

func sortByParentWithChildren(grs []group, id int) []group {
	var children []group
	for i := 0; i < len(grs); i++ {
		if grs[i].ParentID == id {
			children = append(children, grs[i])
			children = append(children, sortByParentWithChildren(grs, grs[i].GroupID)...)
		}
	}
	return children
}

func contains(grs []group, id int) bool {
	for i := 0; i < len(grs); i++ {
		if id == grs[i].GroupID {
			return true
		}
	}
	return false
}

func topParentsHandler(w http.ResponseWriter, r *http.Request) {
	method := r.Method
	if method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var topParents []group
	for i := 0; i < len(taskGroups); i++ {
		if taskGroups[i].ParentID == 0 {
			topParents = append(topParents, taskGroups[i])
		}
	}
	topParents = sortGroupsByName(topParents, 0, len(topParents))
	err := json.NewEncoder(w).Encode(topParents)
	if err != nil {
		log.Fatal(err)
	}
}

func groupsChildrenHandler(w http.ResponseWriter, r *http.Request) {
	method := r.Method
	if method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	vars := mux.Vars(r)
	ID, err := strconv.Atoi(vars["id"])
	if err != nil || !contains(taskGroups, ID) {
		http.NotFound(w, r)
		return
	}
	children := getChildren(taskGroups, ID)
	err = json.NewEncoder(w).Encode(children)
	if err != nil {
		log.Fatal(err)
	}
}

func getChildren(grs []group, id int) []group {
	var children []group
	for i := 0; i < len(grs); i++ {
		if grs[i].ParentID == id {
			children = append(children, grs[i])
		}
	}
	return children
}

func groupsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ID, err := strconv.Atoi(vars["id"])
	if err != nil || !contains(taskGroups, ID) {
		http.NotFound(w, r)
		return
	}
	var gr group
	method := r.Method
	switch method {
	case "GET":
		gr = getGroup(taskGroups, ID)
		err = json.NewEncoder(w).Encode(gr)
		if err != nil {
			log.Fatal(err)
		}
		return
	case "PUT":
		renderTemplate(w, "group", &gr)
	case "DELETE":
		taskGroups, err = removeGroup(taskGroups, ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
}

func getGroup(grs []group, id int) group {
	var gr group
	for i := 0; i < len(grs); i++ {
		if grs[i].GroupID == id {
			gr = grs[i]
			break
		}
	}
	return gr
}

func removeGroup(grs []group, id int) ([]group, error) {
	if getChildren(taskGroups, id) != nil {
		return nil, errors.New("has dependent groups")
	}
	if getTasks(tasks, id) != nil {
		return nil, errors.New("has dependent tasks")
	}
	return grs, nil
}

func getTasks(t []task, id int) []task {
	var newTasks []task
	for i := 0; i < len(t); i++ {
		if t[i].GroupID == id {
			newTasks = append(newTasks, t[i])
		}
	}
	return newTasks
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
		return newTasks
	}
	if lim > len(newTasks) {
		lim = len(newTasks)
	}
	newTasks = newTasks[:lim]
	return newTasks
}

func sortTasksByName(ts []task) []task {
	for i := 1; i < len(ts); i++ {
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
	for i := 1; i < len(ts); i++ {
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

var templates = template.Must(template.ParseFiles("group.html", "task.html"))

func renderTemplate(w http.ResponseWriter, tmpl string, g *group) {
	err := templates.ExecuteTemplate(w, tmpl+".html", g)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/groups", groupsListHandler)
	r.HandleFunc("/groups/top_parents", topParentsHandler)
	r.HandleFunc("/groups/children/{id:[0-9]+}", groupsChildrenHandler)
	r.HandleFunc("/groups/{id:[0-9]+}", groupsHandler)
	r.HandleFunc("/tasks", tasksListHandler)
	http.Handle("/", r)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
