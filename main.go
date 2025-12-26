package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

type Task struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

type API struct {
	tasks  map[int]Task
	nextID int
	mu     sync.Mutex
}

func NewAPI() *API {
	return &API{
		tasks:  make(map[int]Task),
		nextID: 1,
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{
		"error": msg,
	})
}

func (api *API) tasksHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {

	case http.MethodGet:
		api.mu.Lock()
		list := make([]Task, 0, len(api.tasks))
		for _, task := range api.tasks {
			list = append(list, task)
		}
		api.mu.Unlock()
		writeJSON(w, http.StatusOK, list)

	case http.MethodPost:
		var input struct {
			Title string `json:"title"`
		}

		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json")
			return
		}

		api.mu.Lock()
		task := Task{
			ID:        api.nextID,
			Title:     input.Title,
			Completed: false,
		}
		api.tasks[api.nextID] = task
		api.nextID++
		api.mu.Unlock()

		writeJSON(w, http.StatusCreated, task)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (api *API) taskHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/tasks/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	api.mu.Lock()
	task, ok := api.tasks[id]
	api.mu.Unlock()
	if !ok {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}

	switch r.Method {

	case http.MethodGet:
		writeJSON(w, http.StatusOK, task)

	case http.MethodPut:
		var input struct {
			Completed bool `json:"completed"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json")
			return
		}
		api.mu.Lock()
		task.Completed = input.Completed
		api.tasks[id] = task
		api.mu.Unlock()

		writeJSON(w, http.StatusOK, task)

	case http.MethodDelete:
		api.mu.Lock()
		delete(api.tasks, id)
		api.mu.Unlock()

		writeJSON(w, http.StatusOK, map[string]string{
			"status": "deleted",
		})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func main() {
	api := NewAPI()
	http.HandleFunc("/tasks", api.tasksHandler)
	http.HandleFunc("/tasks/", api.taskHandler)

	http.ListenAndServe(":8080", nil)
}
