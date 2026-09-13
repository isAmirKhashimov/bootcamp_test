// Заготовка сервера. Закрывайте этапы по одному, пока не позеленеет go test ./tests/ -v
//
// Запуск: go run ./cmd/server — порт берётся из PORT, по умолчанию 8080.
// Панель уже раздаётся: откройте http://localhost:8080/ и смотрите, как этапы
// зеленеют по ходу работы.
package main

import (
	"container/list"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

var messages = list.New()
var processMessage = getMessageProceessor()

type Message struct {
	Id        int    `json:"id"`
	Content   string `json:"message"`
	CreatedAt string `json:"created_at"`
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()

	// Панель из frontend/. Каталог берётся относительно рабочего, поэтому
	// запускайте из корня модуля: go run ./cmd/server
	mux.Handle("/", http.FileServer(http.Dir("frontend")))
	mux.HandleFunc("GET /health", checkHealth)
	mux.HandleFunc("POST /echo", echoMessage)
	mux.HandleFunc("POST /messages", sendMessage)

	// TODO Этап 1: GET /health           -> 200, тело "ok"
	// TODO Этап 2: POST /echo            -> тело запроса без изменений
	// TODO Этап 3: POST /echo            -> на application/json разобрать {"message": "..."} и вернуть JSON
	// TODO Этап 4: POST /messages        -> сохранить в памяти, 201
	// TODO Этап 5: GET /messages         -> все сообщения, новые сверху
	// TODO Этап 6: DELETE /messages/{id} -> 204, либо 404 если такого нет

	log.Printf("сервер слушает http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func checkHealth(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}

func echoMessage(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to load the request body", http.StatusInternalServerError)
		return
	}

	contentType := r.Header.Get("Content-Type")
	w.Header().Set("Content-Type", contentType)

	if contentType == "application/json" && !json.Valid(bodyBytes) {
		http.Error(w, "Content is not valid", http.StatusBadRequest)
		return
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write(bodyBytes)
	}
}

func sendMessage(w http.ResponseWriter, r *http.Request) {

	var data map[string]string

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	defer r.Body.Close()

	requestMessage, exists := data["message"]
	if !exists || requestMessage == "" {
		http.Error(w, "'message' should not be empty", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	responseMessage, err := processMessage(requestMessage)
	if err != nil {
		http.Error(w, "Cannot process message: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(responseMessage); err != nil {
		http.Error(w, "Cannot serialize message: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func getMessageProceessor() func(string) (Message, error) {
	currentId := 0
	var mu sync.Mutex

	return func(messageText string) (Message, error) {
		mu.Lock()
		defer mu.Unlock()

		currentId++
		now := time.Now().UTC()
		formattedTime := now.Format(time.RFC3339)
		message := Message{
			Id:        currentId,
			Content:   messageText,
			CreatedAt: formattedTime,
		}

		messages.PushFront(message)

		return message, nil
	}
}
