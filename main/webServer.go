package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
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
	TaskID        string `json:"task_id"`
	GroupID       int    `json:"group_id"`
	Task          string `json:"task"`
	Completed     bool   `json:"completed"`
	CreatedDate   string `json:"created_at"`
	CompletedDate string `json:"completed_at"`
}

type statistics struct {
	Completed int
	Created   int
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

func writeGroups(grs []group) {
	groupsFile, err := json.Marshal(grs)
	if err != nil {
		log.Fatal(err)
	}
	err = ioutil.WriteFile("groups.json", groupsFile, 0644)
	if err != nil {
		log.Fatal(err)
	}
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

func writeTasks(ts []task) {
	tasksFile, err := json.Marshal(ts)
	if err != nil {
		log.Fatal(err)
	}
	err = ioutil.WriteFile("tasks.json", tasksFile, 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func groupsListHandler(w http.ResponseWriter, r *http.Request) {
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

func containsGroup(grs []group, id int) bool {
	for i := 0; i < len(grs); i++ {
		if id == grs[i].GroupID {
			return true
		}
	}
	return false
}

func topParentsHandler(w http.ResponseWriter, r *http.Request) {
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
	vars := mux.Vars(r)
	ID, err := strconv.Atoi(vars["id"])
	if err != nil || !containsGroup(taskGroups, ID) {
		http.NotFound(w, r)
		return
	}
	children := getChildren(taskGroups, ID)
	if children == nil {
		http.Error(w, "400 has no children", http.StatusBadRequest)
		return
	}
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

func groupShowHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ID, err := strconv.Atoi(vars["id"])
	if err != nil || !containsGroup(taskGroups, ID) {
		http.NotFound(w, r)
		return
	}
	err = json.NewEncoder(w).Encode(getGroup(taskGroups, ID))
	if err != nil {
		log.Fatal(err)
	}
}

func groupEditHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ID, err := strconv.Atoi(vars["id"])
	if err != nil || !containsGroup(taskGroups, ID) {
		http.NotFound(w, r)
		return
	}
	var gr group
	err = json.NewDecoder(r.Body).Decode(&gr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if containsGroup(taskGroups, gr.GroupID) && gr.GroupID != ID {
		http.Error(w, "400 group with this ID already exists", http.StatusBadRequest)
		return
	}
	if getChildren(taskGroups, ID) != nil && gr.GroupID != ID {
		http.Error(w, "400 has dependent groups", http.StatusBadRequest)
		return
	}
	if getTasksByGroupID(tasks, ID) != nil && gr.GroupID != ID {
		http.Error(w, "400 has dependent tasks", http.StatusBadRequest)
		return
	}
	if !containsGroup(taskGroups, gr.ParentID) && gr.ParentID != 0 {
		http.Error(w, "400 parent with this ID does not exist", http.StatusBadRequest)
		return
	}
	n := getGroupNumByID(taskGroups, ID)
	taskGroups[n] = gr
	err = json.NewEncoder(w).Encode(gr)
}

func getGroupNumByID(grs []group, id int) int {
	var n int
	for i := 0; i < len(grs); i++ {
		if grs[i].GroupID == id {
			n = i
			break
		}
	}
	return n
}

func groupDeleteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ID, err := strconv.Atoi(vars["id"])
	if err != nil || !containsGroup(taskGroups, ID) {
		http.NotFound(w, r)
		return
	}
	taskGroups, err = removeGroup(taskGroups, ID)
	if err != nil {
		http.Error(w, "400 "+err.Error(), http.StatusBadRequest)
	} else {
		_, err = fmt.Fprint(w, "group deleted")
		if err != nil {
			log.Fatal(err)
		}
	}
}

func removeGroup(grs []group, id int) ([]group, error) {
	if getChildren(grs, id) != nil {
		return grs, errors.New("has dependent groups")
	}
	if getTasksByGroupID(tasks, id) != nil {
		return grs, errors.New("has dependent tasks")
	}
	for i := 0; i < len(grs); i++ {
		if grs[i].GroupID == id {
			for g := i; g < len(grs)-1; g++ {
				grs[g] = grs[g+1]
			}
			break
		}
	}
	grs = grs[:len(grs)-1]
	return grs, nil
}

func newGroupHandler(w http.ResponseWriter, r *http.Request) {
	var gr group
	err := json.NewDecoder(r.Body).Decode(&gr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if gr.Name == "" {
		http.Error(w, "400 name is not specified", http.StatusBadRequest)
		return
	}
	if !containsGroup(taskGroups, gr.ParentID) && gr.ParentID != 0 {
		http.Error(w, "400 parent with this ID does not exist", http.StatusBadRequest)
		return
	}
	gr.GroupID = getMaxID(taskGroups) + 1
	taskGroups = append(taskGroups, gr)
	err = json.NewEncoder(w).Encode(gr)
}

func getMaxID(grs []group) int {
	max := 0
	for i := 0; i < len(grs); i++ {
		if grs[i].GroupID > max {
			max = grs[i].GroupID
		}
	}
	return max
}

func getTasksByGroupID(t []task, id int) []task {
	var newTasks []task
	for i := 0; i < len(t); i++ {
		if t[i].GroupID == id {
			newTasks = append(newTasks, t[i])
		}
	}
	return newTasks
}

func tasksListHandler(w http.ResponseWriter, r *http.Request) {
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

func newTaskHandler(w http.ResponseWriter, r *http.Request) {
	var t task
	err := json.NewDecoder(r.Body).Decode(&t)
	if err != nil {
		http.Error(w, "400 "+err.Error(), http.StatusBadRequest)
		return
	}
	if t.Task == "" {
		http.Error(w, "400 task is not specified", http.StatusBadRequest)
		return
	}
	if !containsGroup(taskGroups, t.GroupID) {
		http.Error(w, "400 group with this ID does not exist", http.StatusBadRequest)
		return
	}
	hash := sha1.New()
	hash.Write([]byte(t.Task))
	t.TaskID = hex.EncodeToString(hash.Sum(nil))[:5]
	if containsTask(tasks, t.TaskID) {
		http.Error(w, "400 task with this ID already exists", http.StatusBadRequest)
		return
	}
	t.CreatedDate = time.Now().Format(time.RFC3339Nano)
	tasks = append(tasks, t)
	err = json.NewEncoder(w).Encode(t)
	if err != nil {
		log.Fatal(err)
	}
}

func groupTasksHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ID, err := strconv.Atoi(vars["id"])
	if err != nil || !containsGroup(taskGroups, ID) {
		http.NotFound(w, r)
		return
	}
	newTasks := getTasksByGroupID(tasks, ID)
	if newTasks == nil {
		http.Error(w, "400 has no dependent tasks", http.StatusBadRequest)
		return
	}
	t := r.URL.Query().Get("type")
	switch t {
	case "completed":
		newTasks = getCompletedTasks(newTasks)
	case "working":
		newTasks = getWorkingTasks(newTasks)
	}
	if len(newTasks) == 0 {
		http.Error(w, "400 has no dependent tasks of this type", http.StatusBadRequest)
		return
	}
	err = json.NewEncoder(w).Encode(newTasks)
	if err != nil {
		log.Fatal(err)
	}
}

func taskHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if !containsTask(tasks, vars["id"]) {
		http.NotFound(w, r)
		return
	}
	n := getTaskNumByID(tasks, vars["id"])
	f := r.URL.Query().Get("finished")
	var t task
	var err error
	switch f {
	case "true":
		tasks, err = changeTaskType(tasks, vars["id"], true)
	case "false":
		tasks, err = changeTaskType(tasks, vars["id"], false)
	case "":
		err = json.NewDecoder(r.Body).Decode(&t)
		if err != nil {
			http.Error(w, "400 "+err.Error(), http.StatusBadRequest)
			return
		}
		if t.Task == "" {
			http.Error(w, "400 task is not specified", http.StatusBadRequest)
			return
		}
		if !containsGroup(taskGroups, t.GroupID) {
			http.Error(w, "400 group with this ID does not exist", http.StatusBadRequest)
			return
		}
		hash := sha1.New()
		hash.Write([]byte(t.Task))
		t.TaskID = hex.EncodeToString(hash.Sum(nil))[:5]
		if containsTask(tasks, t.TaskID) {
			http.Error(w, "400 task with this ID already exists", http.StatusBadRequest)
			return
		}
		t.Completed = tasks[n].Completed
		t.CreatedDate = tasks[n].CreatedDate
		t.CompletedDate = tasks[n].CompletedDate
		tasks[n] = t
	default:
		http.Error(w, "400 bad request", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = json.NewEncoder(w).Encode(tasks[n])
	if err != nil {
		log.Fatal(err)
	}
}

func getTaskNumByID(ts []task, id string) int {
	var n int
	for i := 0; i < len(ts); i++ {
		if ts[i].TaskID == id {
			n = i
			break
		}
	}
	return n
}

func changeTaskType(ts []task, id string, t bool) ([]task, error) {
	for i := 0; i < len(ts); i++ {
		if ts[i].TaskID == id {
			if t == ts[i].Completed {
				return ts, errors.New("400 already of this type")
			}
			ts[i].Completed = t
			if t {
				ts[i].CompletedDate = time.Now().Format(time.RFC3339Nano)
			} else {
				ts[i].CompletedDate = ""
			}
			break
		}
	}
	return ts, nil
}

func containsTask(ts []task, id string) bool {
	for i := 0; i < len(ts); i++ {
		if ts[i].TaskID == id {
			return true
		}
	}
	return false
}

func statHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	stat, err := getStat(tasks, vars["period"])
	if err != nil {
		http.NotFound(w, r)
		return
	}
	err = json.NewEncoder(w).Encode(stat)
	if err != nil {
		log.Fatal(err)
	}
}

func getStat(ts []task, period string) (statistics, error) {
	var s statistics
	n := time.Now()
	var periodStart time.Time
	var periodEnd time.Time
	switch period {
	case "today":
		periodStart = n.Add(-time.Hour)
		periodStart = time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, n.Location())
		periodEnd = n
	case "yesterday":
		periodStart = time.Date(n.Year(), n.Month(), n.Day()-1, 0, 0, 0, 0, n.Location())
		periodEnd = time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, n.Location())
	case "week":
		periodStart = time.Date(n.Year(), n.Month(), n.Day()-7, n.Hour(), n.Minute(), n.Second(), n.Nanosecond(), n.Location())
		periodEnd = n
	case "month":
		periodStart = time.Date(n.Year(), n.Month()-1, n.Day(), n.Hour(), n.Minute(), n.Second(), n.Nanosecond(), n.Location())
		periodEnd = n
	default:
		return statistics{0, 0}, errors.New("not found")
	}
	var createdDate time.Time
	var completedDate time.Time
	var err error
	for i := 0; i < len(ts); i++ {
		createdDate, err = time.Parse(time.RFC3339Nano, ts[i].CreatedDate)
		if err != nil {
			log.Fatal(err)
		}
		if ts[i].CompletedDate == "" {
			completedDate = n
		} else {
			completedDate, err = time.Parse(time.RFC3339Nano, ts[i].CompletedDate)
		}
		if err != nil {
			log.Fatal(err)
		}
		if createdDate.Before(periodEnd) && createdDate.After(periodStart) {
			s.Created++
		}
		if completedDate.Before(periodEnd) && completedDate.After(periodStart) {
			s.Completed++
		}
	}
	return s, nil
}

func main() {
	var wait time.Duration
	flag.DurationVar(&wait, "graceful-timeout", time.Second*15, "the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
	flag.Parse()
	r := mux.NewRouter()
	r.HandleFunc("/groups", groupsListHandler).Methods("GET")
	r.HandleFunc("/groups/top_parents", topParentsHandler).Methods("GET")
	r.HandleFunc("/groups/children/{id:[0-9]+}", groupsChildrenHandler).Methods("GET")
	r.HandleFunc("/groups/new", newGroupHandler).Methods("POST")
	r.HandleFunc("/groups/{id:[0-9]+}", groupShowHandler).Methods("GET")
	r.HandleFunc("/groups/{id:[0-9]+}", groupEditHandler).Methods("PUT")
	r.HandleFunc("/groups/{id:[0-9]+}", groupDeleteHandler).Methods("DELETE")
	r.HandleFunc("/tasks", tasksListHandler).Methods("GET")
	r.HandleFunc("/tasks/new", newTaskHandler).Methods("POST")
	r.HandleFunc("/tasks/group/{id:[0-9]+}", groupTasksHandler).Methods("GET")
	r.HandleFunc("/tasks/{id:[a-zA-Z0-9]+}", taskHandler).Methods("PUT")
	r.HandleFunc("/stat/{period}", statHandler).Methods("GET")
	http.Handle("/", r)
	srv := &http.Server{
		Addr:         "0.0.0.0:8080",
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      r,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	err := srv.Shutdown(ctx)
	if err != nil {
		log.Fatal(err)
	}
	writeGroups(taskGroups)
	writeTasks(tasks)
	log.Println("shutting down")
	os.Exit(0)
}
