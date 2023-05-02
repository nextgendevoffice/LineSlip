package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/line/line-bot-sdk-go/linebot"
)

var (
	bot          *linebot.Client
	memberSystem *MemberSystem
)

type TransactionDetails struct {
	Amount        string `json:"amount"`
	FromBank      string `json:"from_bank"`
	Sender        string `json:"sender"`
	Receiver      string `json:"receiver"`
	Timestamp     string `json:"timestamp"`
	TransactionID string `json:"transaction_id"`
}
type APIErrorResponse struct {
	ErrorCode int `json:"error_code"`
}

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

func fetchTransactionDetails(qrString string) (*TransactionDetails, error) {
	url := fmt.Sprintf("https://fast888.co/api/get_tr_detail/%s", qrString)
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	// Log the response body
	log.Println("Response from API:", string(body))

	// Check if the response contains an error
	var errorResponse APIErrorResponse
	err = json.Unmarshal(body, &errorResponse)
	if err == nil && errorResponse.ErrorCode != 0 {
		return nil, fmt.Errorf("API error, error code: %d", errorResponse.ErrorCode)
	}

	var details TransactionDetails
	err = json.Unmarshal(body, &details)
	if err != nil {
		return nil, err
	}

	return &details, nil
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
		// Fetch transaction details using the decoded QR code string
		details, err := fetchTransactionDetails(decodedString)
		if err != nil {
			log.Println("Error fetching transaction details:", err)
			replyText(event.ReplyToken, "Failed to fetch transaction details. Please try again later.")
			return
		}

		// Format the transaction details as a readable message
		message := fmt.Sprintf(
			"ยอดโอน: %s บาท\nโอนจากธนาคาร: \nผู้โอน : %s\nผู้รับเงิน: %s\nเวลา: %s\nเลขที่ : %s",
			details.Amount,
			details.Sender,
			details.Receiver,
			details.Timestamp,
			details.TransactionID,
		)

		replyText(event.ReplyToken, message)
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
