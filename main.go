package main

import (
	"encoding/json"
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

	if message.Text == "/join" {
		if memberSystem.IsMember(userID) {
			replyText(event.ReplyToken, "คุณเป็นสมาชิกอยู่แล้วครับ")
		} else {
			memberSystem.AddMember(userID)
			log.Printf("User %s joined", userID)
			replyText(event.ReplyToken, "คุณเข้าร่วมเป็นที่เรียบร้อย คุณสามารถส่งสลิปเพื่อเช็คได้เลยค่ะ")
		}
		return
	}

	if !memberSystem.IsMember(userID) {
		log.Printf("User %s is not a member", userID)
		replyText(event.ReplyToken, "กรุณาใช้คำสั่ง /join เพื่อใช้งานเช็คสลิป")
		return
	}

	replyText(event.ReplyToken, "กรุณาส่งสลิปเพื่อเช็คสลิปได้เลยค่ะ")
}

func fetchDataFromAPI(input string) (string, error) {
	apiURL := fmt.Sprintf("https://fast888.co/api/get_tr_detail/%s", input)
	resp, err := http.Get(apiURL)
	if err != nil {
		fmt.Printf("Error sending request to API: %v\n", err)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("API returned non-200 status code: %d\n", resp.StatusCode)
		return "", fmt.Errorf("API returned non-200 status code: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		fmt.Printf("Error decoding API response: %v\n", err)
		return "", err
	}

	// Extract the desired information from the result map
	// For example, if you want to extract the field "data"
	data, ok := result["data"].(string)
	if !ok {
		fmt.Printf("Error extracting data field from API response: %v\n", result)
		return "", fmt.Errorf("Error extracting data field from API response")
	}

	return data, nil
}

func handleImageMessage(event *linebot.Event, message *linebot.ImageMessage) {
	userID := event.Source.UserID

	if !memberSystem.IsMember(userID) {
		replyText(event.ReplyToken, "กรุณาใช้คำสั่ง /join เพื่อใช้งานเช็คสลิป")
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
		var apiResult string
		apiResult, err = fetchDataFromAPI(decodedString)
		if err != nil {
			replyText(event.ReplyToken, "Error fetching data from API")
		} else {
			replyText(event.ReplyToken, fmt.Sprintf("Decoded content: %s\nAPI Result: %s", decodedString, apiResult))
		}
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
