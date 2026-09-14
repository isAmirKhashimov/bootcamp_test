package main

import (
	"cmp"
	"slices"
	"strconv"
	"sync"
	"time"
)

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
