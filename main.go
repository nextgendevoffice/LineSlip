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

func fetchTransactionDetails(qrCode string) (*TransactionDetails, error) {
	url := fmt.Sprintf("https://fast888.co/api/get_tr_detail/%s", qrCode)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("authority", "fast888.co")
	req.Header.Set("accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")
	req.Header.Set("accept-language", "en-US,en;q=0.9")
	req.Header.Set("cache-control", "max-age=0")
	req.Header.Set("cookie", "fast888=7gp6ko62gprvim8oooimjcc1toqcq5lh")
	req.Header.Set("sec-ch-ua", `"Chromium";v="112", "Microsoft Edge";v="112", "Not:A-Brand";v="99"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"Windows"`)
	req.Header.Set("sec-fetch-dest", "document")
	req.Header.Set("sec-fetch-mode", "navigate")
	req.Header.Set("sec-fetch-site", "none")
	req.Header.Set("sec-fetch-user", "?1")
	req.Header.Set("upgrade-insecure-requests", "1")
	req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/112.0.0.0 Safari/537.36 Edg/112.0.1722.64")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching transaction details: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		log.Printf("Response from API: %s", string(bodyBytes))
		return nil, fmt.Errorf("error fetching transaction details: status code %d", resp.StatusCode)
	}

	var details TransactionDetails
	if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
		return nil, fmt.Errorf("error decoding transaction details: %v", err)
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
			replyText(event.ReplyToken, "You are already a member")
		} else {
			memberSystem.AddMember(userID)
			log.Printf("User %s joined", userID)
			replyText(event.ReplyToken, "You are now a member")
		}
		return
	}

	if !memberSystem.IsMember(userID) {
		log.Printf("User %s is not a member", userID)
		replyText(event.ReplyToken, "กรุณาใช้คำสั่ง /join เพื่อใช้งานเช็คสลิป")
		return
	}

	if message.Text == "/join" {
		memberSystem.AddMember(userID)
		log.Printf("User %s joined", userID)
		replyText(event.ReplyToken, "You are now a member")
		return
	}

	replyText(event.ReplyToken, "กรุณาส่งสลิปเพื่อเช็คสลิปได้เลยค่ะ")
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
