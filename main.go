package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/line/line-bot-sdk-go/linebot"
)

var (
	bot          *linebot.Client
	memberSystem *MemberSystem
)

func main() {
	memberSystem = NewMemberSystem()
	var err error
	bot, err = linebot.New(os.Getenv("LINE_CHANNEL_SECRET"), os.Getenv("LINE_CHANNEL_TOKEN"))
	if err != nil {
		fmt.Println("Error initializing linebot:", err)
		return
	}

	memberSystem = NewMemberSystem()

	http.HandleFunc("/callback", handleCallback)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Println("Listening on port", port)
	http.ListenAndServe(":"+port, nil)
}

func handleCallback(w http.ResponseWriter, r *http.Request) {
	events, err := bot.ParseRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Println("Error parsing request:", err)
		return
	}

	for _, event := range events {
		switch event.Type {
		case linebot.EventTypeMessage:
			handleMessage(event)

		case linebot.EventTypeFollow:
			userID := event.Source.UserID
			memberSystem.AddMember(userID)
			log.Printf("User %s followed the account and was added as a member", userID)

		case linebot.EventTypePostback:
			handlePostback(event)
		}
	}
}

func handleMessage(event *linebot.Event) {
	switch message := event.Message.(type) {
	case *linebot.TextMessage:
		handleTextMessage(event, message)
	case *linebot.ImageMessage:
		handleImageMessage(event, message)
	}
}

func handleTextMessage(event *linebot.Event, message *linebot.TextMessage) {
	userID := event.Source.UserID

	if !memberSystem.IsMember(userID) {
		log.Printf("User %s is not a member", userID)
		replyText(event.ReplyToken, "Please join by sending /join command")
		return
	}

	if message.Text == "/join" {
		memberSystem.AddMember(userID)
		log.Printf("User %s joined", userID)
		replyText(event.ReplyToken, "You are now a member")
		return
	}

	replyText(event.ReplyToken, "Please send a QR code image to decode")
}

func handleImageMessage(event *linebot.Event, message *linebot.ImageMessage) {
	userID := event.Source.UserID

	if !memberSystem.IsMember(userID) {
		replyText(event.ReplyToken, "Please join by sending /join command")
		return
	}

	response, err := bot.GetMessageContent(message.ID).Do()
	if err != nil {
		fmt.Println("Error getting message content:", err)
		return
	}
	defer response.Content.Close()

	filePath := fmt.Sprintf("%s.png", message.ID)
	file, err := os.Create(filePath)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer os.Remove(filePath)

	_, err = io.Copy(file, response.Content)
	file.Close()
	if err != nil {
		fmt.Println("Error saving file:", err)
		return
	}

	decodedString, err := DecodeQRCode(filePath)
	if err != nil {
		replyText(event.ReplyToken, "Error decoding QR code")
	} else {
		replyText(event.ReplyToken, fmt.Sprintf("Decoded content: %s", decodedString))
	}
}

func handlePostback(event *linebot.Event) {
	// Add any postback handling if needed
}

func replyText(replyToken, text string) {
	if _, err := bot.ReplyMessage(replyToken, linebot.NewTextMessage(text)).Do(); err != nil {
		fmt.Println("Error sending reply message:", err)
	}
}
