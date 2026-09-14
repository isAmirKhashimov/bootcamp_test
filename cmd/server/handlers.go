package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

// Step 1: GET /health
func (_ *Server) checkHealth(w http.ResponseWriter, r *http.Request) {

	printLog := getLogger("GET /health")
	printLog("Health check started")
	defer printLog("Health check completed")

	w.Write([]byte("ok"))
}

// Step 2, 3: POST /echo
func (_ *Server) echoMessage(w http.ResponseWriter, r *http.Request) {
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
func (server *Server) sendMessage(w http.ResponseWriter, r *http.Request) {
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

	responseMessage, err := server.messageProcessor.AddMessage(requestMessage)

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
func (server *Server) getMessages(w http.ResponseWriter, r *http.Request) {
	printLog := getLogger("GET /messages")
	printLog("Getting message started")
	defer printLog("Getting message completed")

	w.Header().Set("Content-Type", "application/json")

	messages := server.messageProcessor.GetMessagesReverse()
	if err := json.NewEncoder(w).Encode(&messages); err != nil {
		printLog("Cannot serialize message: " + err.Error() + ". Return 500")
		http.Error(w, "Cannot serialize message: "+err.Error(), http.StatusInternalServerError)
		return
	}

	printLog("Successfully serialized messages. Messages count: " + strconv.Itoa(len(messages)))
}

// Step 6: DELETE /messages/{id}
func (server *Server) deleteMessage(w http.ResponseWriter, r *http.Request) {
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

	if !server.messageProcessor.TryDeleteMessageById(num) {
		printLog("Target message does not exist. Return 404")
		http.Error(w, "Target message does not exist", http.StatusNotFound)
		return
	}

	printLog("Message deleted successully. Return 204")
	w.WriteHeader(http.StatusNoContent)
}
