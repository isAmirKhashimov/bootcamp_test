// Заготовка сервера. Закрывайте этапы по одному, пока не позеленеет go test ./tests/ -v
//
// Запуск: go run ./cmd/server — порт берётся из PORT, по умолчанию 8080.
// Панель уже раздаётся: откройте http://localhost:8080/ и смотрите, как этапы
// зеленеют по ходу работы.
package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

// List I used before cannot be simply serialized as complex structure, so I decided to proceed with slices and simple reverse
var (
	messages       []Message
	messagesMu     sync.Mutex
	processMessage = getMessageProceessor()
)

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
	mux.HandleFunc("GET /messages", getMessages)

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
	log.Printf("[GET /health] Health check started")
	defer log.Printf("[GET /health] Health check completed")

	w.Write([]byte("ok"))
}

func echoMessage(w http.ResponseWriter, r *http.Request) {
	log.Printf("[POST /echo] Echoing message started")
	defer log.Printf("[POST /echo] Echoing message completed")

	defer r.Body.Close()

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[POST /echo] Failed to load the request body. Return 500")
		http.Error(w, "Failed to load the request body", http.StatusInternalServerError)
		return
	}

	contentType := r.Header.Get("Content-Type")
	w.Header().Set("Content-Type", contentType)

	if contentType == "application/json" && !json.Valid(bodyBytes) {
		log.Printf("[POST /echo] Request content is not valid. Return 400")
		http.Error(w, "Request content is not valid", http.StatusBadRequest)
		return
	} else {
		log.Printf("[POST /echo] Successfully read request body: \"%v\". Return 200", string(bodyBytes))
		w.WriteHeader(http.StatusOK)
		w.Write(bodyBytes)
	}
}

func sendMessage(w http.ResponseWriter, r *http.Request) {
	log.Printf("[POST /messages] Sending message started")
	defer log.Printf("[POST /messages] Sending message completed")

	var data map[string]string

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		log.Printf("[POST /messages] Request content is not valid. Return 400")
		http.Error(w, "Request content is not valid", http.StatusBadRequest)
		return
	}

	defer r.Body.Close()

	requestMessage, exists := data["message"]
	if !exists || requestMessage == "" {
		log.Printf("[POST /messages] 'message' should not be empty. Return 400")
		http.Error(w, "'message' should not be empty", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	responseMessage, err := processMessage(requestMessage)

	if err != nil {
		log.Printf("[POST /messages] Cannot process message: %v. Return 500", err.Error())
		http.Error(w, "Cannot process message: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("[POST /messages] Message processed successully: \"%v\". Return 201", responseMessage)
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(responseMessage); err != nil {
		log.Printf("[POST /messages] Cannot serialize message: %v. Return 500", err.Error())
		http.Error(w, "Cannot serialize message: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func getMessages(w http.ResponseWriter, r *http.Request) {
	log.Printf("[GET /messages] Getting messages started")
	defer log.Printf("[GET /messages] Getting messages completed")

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(getMessagesReverse()); err != nil {
		log.Printf("[GET /messages] Cannot serialize message: %v. Return 500", err.Error())
		http.Error(w, "Cannot serialize message: "+err.Error(), http.StatusInternalServerError)
		return
	}

	defer log.Printf("[GET /messages] Successfully serialized messages. Messages count: %v", len(messages))
}

func getMessagesReverse() []Message {
	messagesMu.Lock()
	defer messagesMu.Unlock()

	log.Printf("[getMessagesReverse] Message processing started")
	defer log.Printf("[getMessagesReverse] Message processing completed")

	n := len(messages)
	if n == 0 {
		return make([]Message, 0)
	}

	result := make([]Message, n)

	for i, msg := range messages {
		result[n-1-i] = msg
	}

	return result
}

func getMessageProceessor() func(string) (Message, error) {
	currentId := 0
	log.Printf("[getMessageProcessor] Message processor initialized, ID = %v", currentId)

	return func(messageText string) (Message, error) {
		messagesMu.Lock()
		defer messagesMu.Unlock()

		currentId++

		log.Printf("[getMessageProcessor] Message processing started, ID = %v", currentId)
		defer log.Printf("[getMessageProcessor] Message processing completed")

		now := time.Now().UTC()
		formattedTime := now.Format(time.RFC3339)
		message := Message{
			Id:        currentId,
			Content:   messageText,
			CreatedAt: formattedTime,
		}

		messages = append(messages, message)

		return message, nil
	}
}
