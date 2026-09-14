// Заготовка сервера. Закрывайте этапы по одному, пока не позеленеет go test ./tests/ -v
//
// Запуск: go run ./cmd/server — порт берётся из PORT, по умолчанию 8080.
// Панель уже раздаётся: откройте http://localhost:8080/ и смотрите, как этапы
// зеленеют по ходу работы.
package main

import (
	"cmp"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"slices"
	"strconv"
	"sync"
	"time"
)

// List I used before cannot be simply serialized as complex structure, so I decided to proceed with slices and simple reverse
var messageProcessor IMessageProcessor = &MessageProcessor{}

type Message struct {
	Id        int    `json:"id"`
	Content   string `json:"message"`
	CreatedAt string `json:"created_at"`
}

type IMessageProcessor interface {
	TryDeleteMessageById(id int) bool
	GetMessagesReverse() []Message
	AddMessage(messageText string) (Message, error)
}

type MessageProcessor struct {
	messages   []Message
	messagesMu sync.Mutex
	currentId  int
}

func (messageProcessor *MessageProcessor) TryDeleteMessageById(id int) bool {
	messageProcessor.messagesMu.Lock()
	defer messageProcessor.messagesMu.Unlock()

	printLog := getLogger("MessageProcessor.TryDeleteMessageById")
	printLog("Message deletion started")
	defer printLog("Message deletion completed")

	targetIdx, found := slices.BinarySearchFunc(messageProcessor.messages, id, func(msg Message, targetId int) int {
		return cmp.Compare(msg.Id, targetId)
	})

	if !found {
		printLog("Message was not found")
		return false
	}

	// Here it'd be better to have map or lazy deletion instead of slice.
	printLog("Message was found by index" + strconv.Itoa(targetIdx))
	copy(messageProcessor.messages[targetIdx:], messageProcessor.messages[targetIdx+1:])
	messageProcessor.messages = messageProcessor.messages[:len(messageProcessor.messages)-1]

	return true
}

func (messageProcessor *MessageProcessor) GetMessagesReverse() []Message {
	messageProcessor.messagesMu.Lock()
	defer messageProcessor.messagesMu.Unlock()

	printLog := getLogger("MessageProcessor.GetMessagesReverse")
	printLog("Messages reversing started")
	defer printLog("Messages reversing completed")

	n := len(messageProcessor.messages)
	if n == 0 {
		return make([]Message, 0)
	}

	result := make([]Message, n)

	for i, msg := range messageProcessor.messages {
		result[n-1-i] = msg
	}

	return result
}

func (messageProcessor *MessageProcessor) AddMessage(messageText string) (Message, error) {
	messageProcessor.messagesMu.Lock()
	defer messageProcessor.messagesMu.Unlock()

	messageProcessor.currentId++

	printLog := getLogger("MessageProcessor.AddMessage")
	printLog("Messages reversing started")
	printLog("Message processing started, ID = " + strconv.Itoa(messageProcessor.currentId))
	defer printLog("Message processing completed, ID = " + strconv.Itoa(messageProcessor.currentId))

	now := time.Now().UTC()
	formattedTime := now.Format(time.RFC3339)
	message := Message{
		Id:        messageProcessor.currentId,
		Content:   messageText,
		CreatedAt: formattedTime,
	}

	messageProcessor.messages = append(messageProcessor.messages, message)

	return message, nil
}

func getLogger(header string) func(string) {
	return func(messageText string) {
		log.Printf("[%v]: %v", header, messageText)
	}
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
	mux.HandleFunc("DELETE /messages/{id}", deleteMessage)

	// TODO Этап 1: GET /health           -> 200, тело "ok"
	// TODO Этап 2: POST /echo            -> тело запроса без изменений
	// TODO Этап 3: POST /echo            -> на application/json разобрать {"message": "..."} и вернуть JSON
	// TODO Этап 4: POST /messages        -> сохранить в памяти, 201
	// TODO Этап 5: GET /messages         -> все сообщения, новые сверху
	// TODO Этап 6: DELETE /messages/{id} -> 204, либо 404 если такого нет

	log.Printf("сервер слушает http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

// Step 1: GET /health
func checkHealth(w http.ResponseWriter, r *http.Request) {

	printLog := getLogger("GET /health")
	printLog("Health check started")
	defer printLog("Health check completed")

	w.Write([]byte("ok"))
}

// Step 2, 3: POST /echo
func echoMessage(w http.ResponseWriter, r *http.Request) {
	printLog := getLogger("POST /echo")
	printLog("Echoing message started")
	defer printLog("Echoing message completed")

	defer r.Body.Close()

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		printLog("Failed to load the request body. Return 500")
		http.Error(w, "Failed to load the request body", http.StatusInternalServerError)
		return
	}

	contentType := r.Header.Get("Content-Type")
	w.Header().Set("Content-Type", contentType)

	if contentType == "application/json" && !json.Valid(bodyBytes) {
		printLog("Request content is not valid. Return 400")
		http.Error(w, "Request content is not valid", http.StatusBadRequest)
		return
	} else {
		printLog("Successfully read request body: \"" + string(bodyBytes) + "\". Return 200")
		w.WriteHeader(http.StatusOK)
		w.Write(bodyBytes)
	}
}

// Step 4: POST /messages
func sendMessage(w http.ResponseWriter, r *http.Request) {
	printLog := getLogger("POST /messages")
	printLog("Sending message started")
	defer printLog("Sending message completed")

	var data map[string]string

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		printLog("Request content is not valid. Return 400")
		http.Error(w, "Request content is not valid", http.StatusBadRequest)
		return
	}

	defer r.Body.Close()

	requestMessage, exists := data["message"]
	if !exists || requestMessage == "" {
		printLog("'message' should not be empty. Return 400")
		http.Error(w, "'message' should not be empty", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	responseMessage, err := messageProcessor.AddMessage(requestMessage)

	if err != nil {
		printLog("Cannot process message: " + err.Error() + ". Return 500")
		http.Error(w, "Cannot process message: "+err.Error(), http.StatusInternalServerError)
		return
	}

	printLog("Message processed successully: \"" + fmt.Sprintf("%#v", responseMessage) + "\". Return 201")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(responseMessage); err != nil {
		printLog("Cannot serialize message: " + err.Error() + ". Return 500")
		http.Error(w, "Cannot serialize message: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

// Step 5: GET /messages
func getMessages(w http.ResponseWriter, r *http.Request) {
	printLog := getLogger("GET /messages")
	printLog("Getting message started")
	defer printLog("Getting message completed")

	w.Header().Set("Content-Type", "application/json")

	messages := messageProcessor.GetMessagesReverse()
	if err := json.NewEncoder(w).Encode(&messages); err != nil {
		printLog("Cannot serialize message: " + err.Error() + ". Return 500")
		http.Error(w, "Cannot serialize message: "+err.Error(), http.StatusInternalServerError)
		return
	}

	printLog("Successfully serialized messages. Messages count: " + strconv.Itoa(len(messages)))
}

// Step 6: DELETE /messages/{id}
func deleteMessage(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")

	printLog := getLogger("DELETE /messages/" + idStr)
	printLog("Deleting messages started")
	defer printLog("Getting messages completed")

	w.Header().Set("Content-Type", "application/json")

	num, err := strconv.Atoi(idStr)
	if err != nil {
		printLog("Cannot convert id = " + err.Error() + " to integer. Return 500")
		http.Error(w, "Cannot convert id = \""+idStr+"\" to integer: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if !messageProcessor.TryDeleteMessageById(num) {
		printLog("Target message does not exist. Return 404")
		http.Error(w, "Target message does not exist", http.StatusNotFound)
		return
	}

	printLog("Message deleted successully. Return 204")
	w.WriteHeader(http.StatusNoContent)
}
