package main

import (
	"log"
)

func getLogger(header string) func(string) {
	return func(messageText string) {
		log.Printf("[%v]: %v", header, messageText)
	}
}
