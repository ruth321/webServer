package main

import (
	"context"
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
	method := r.Method
	if method != "GET" {
		http.Error(w, "405 method not allowed", http.StatusMethodNotAllowed)
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

func containsGroup(grs []group, id int) bool {
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
		http.Error(w, "405 method not allowed", http.StatusMethodNotAllowed)
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
		http.Error(w, "405 method not allowed", http.StatusMethodNotAllowed)
		return
	}
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

func groupsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ID, err := strconv.Atoi(vars["id"])
	if err != nil || !containsGroup(taskGroups, ID) {
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
	case "DELETE":
		taskGroups, err = removeGroup(taskGroups, ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else {
			_, err = fmt.Fprint(w, "Group deleted")
			if err != nil {
				log.Fatal(err)
			}
		}
	default:
		http.Error(w, "405 method not allowed", http.StatusMethodNotAllowed)
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
	method := r.Method
	if method != "GET" {
		http.Error(w, "405 method not allowed", http.StatusMethodNotAllowed)
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

func groupTasksHandler(w http.ResponseWriter, r *http.Request) {
	method := r.Method
	if method != "GET" {
		http.Error(w, "405 method not allowed", http.StatusMethodNotAllowed)
		return
	}
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
	method := r.Method
	if method != "PUT" {
		http.Error(w, "405 method not allowed", http.StatusMethodNotAllowed)
		return
	}
	vars := mux.Vars(r)
	if !containsTask(tasks, vars["id"]) {
		http.NotFound(w, r)
		return
	}
	f := r.URL.Query().Get("finished")
	var err error
	switch f {
	case "true":
		tasks, err = changeTaskType(tasks, vars["id"], true)
	case "false":
		tasks, err = changeTaskType(tasks, vars["id"], false)
	case "":
	default:
		http.Error(w, "400 bad request", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	t := getTask(tasks, vars["id"])
	err = json.NewEncoder(w).Encode(t)
	if err != nil {
		log.Fatal(err)
	}
}

func getTask(ts []task, id string) task {
	var t task
	for i := 0; i < len(ts); i++ {
		if ts[i].TaskID == id {
			t = ts[i]
			break
		}
	}
	return t
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
	method := r.Method
	if method != "GET" {
		http.Error(w, "405 method not allowed", http.StatusMethodNotAllowed)
		return
	}
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
	r.HandleFunc("/groups", groupsListHandler)
	r.HandleFunc("/groups/top_parents", topParentsHandler)
	r.HandleFunc("/groups/children/{id:[0-9]+}", groupsChildrenHandler)
	r.HandleFunc("/groups/{id:[0-9]+}", groupsHandler)
	r.HandleFunc("/tasks", tasksListHandler)
	r.HandleFunc("/tasks/group/{id:[0-9]+}", groupTasksHandler)
	r.HandleFunc("/tasks/{id:[a-zA-Z0-9]+}", taskHandler)
	r.HandleFunc("/stat/{period}", statHandler)
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
