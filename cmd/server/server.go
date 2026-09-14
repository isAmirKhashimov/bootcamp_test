package main

import (
	"net/http"
)

type Server struct {
	messageProcessor IMessageProcessor
}

func newServer() http.Handler {
	server := &Server{
		messageProcessor: &MessageProcessor{},
	}

	mux := http.NewServeMux()

	// Панель из frontend/. Каталог берётся относительно рабочего, поэтому
	// запускайте из корня модуля: go run ./cmd/server
	mux.Handle("/", http.FileServer(http.Dir("frontend")))
	mux.HandleFunc("GET /health", server.checkHealth)
	mux.HandleFunc("POST /echo", server.echoMessage)
	mux.HandleFunc("POST /messages", server.sendMessage)
	mux.HandleFunc("GET /messages", server.getMessages)
	mux.HandleFunc("DELETE /messages/{id}", server.deleteMessage)
	return mux
}
