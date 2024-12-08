package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sync"
)

type DeleteRequest struct {
	Task string `json:"task"`
}

var (
	todoList = []string{}
	mu       sync.Mutex
	dataFile = "todos.json"
)

// Save todos to file
func saveTodosToFile() error {
	file, err := os.Create(dataFile)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewEncoder(file).Encode(todoList)
}

// Load todos from file
func loadTodosFromFile() error {
	file, err := os.Open(dataFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()
	return json.NewDecoder(file).Decode(&todoList)
}

func removeFromList(list []string, item string) []string {
	for i, v := range list {
		if v == item {
			return append(list[:i], list[i+1:]...)
		}
	}
	return list
}

func todoHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	switch r.Method {
	case http.MethodGet:
		if r.URL.Query().Get("partial") == "true" {
			partialTmpl, err := template.ParseFiles("templates/partial.html")
			if err != nil {
				http.Error(w, "Could not load partial template", http.StatusInternalServerError)
				log.Println("Error parsing partial template:", err)
				return
			}
			w.Header().Set("Content-Type", "text/html")
			if err := partialTmpl.Execute(w, todoList); err != nil {
				http.Error(w, "Could not render partial template", http.StatusInternalServerError)
				log.Println("Error rendering partial template:", err)
				return
			}
			return
		}

		tmpl, err := template.ParseFiles("templates/index.html")
		if err != nil {
			http.Error(w, "Could not load template", http.StatusInternalServerError)
			log.Println("Error parsing template:", err)
			return
		}
		if err := tmpl.Execute(w, todoList); err != nil {
			http.Error(w, "Could not render template", http.StatusInternalServerError)
			log.Println("Error rendering template:", err)
			return
		}

	case http.MethodPost:
		if task := r.FormValue("task"); task != "" {
			todoList = append(todoList, task)
			if err := saveTodosToFile(); err != nil {
				log.Printf("Error saving todos: %v", err)
			}
			w.WriteHeader(http.StatusOK)
		}

	case http.MethodDelete:
		var req DeleteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		todoList = removeFromList(todoList, req.Task)
		if err := saveTodosToFile(); err != nil {
			log.Printf("Error saving todos: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}
}

func main() {
	// Load saved todos when starting
	if err := loadTodosFromFile(); err != nil {
		log.Printf("Error loading todos: %v", err)
	}

	http.HandleFunc("/", todoHandler)
	port := ":8080"
	fmt.Println("Starting server on port", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal(err)
	}
}
