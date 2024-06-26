package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type InEvent struct {
	Window int `json:"window"`
	Middle int `json:"middle"`
	Aisle  int `json:"aisle"`
}

type OutEvent struct {
	Status int `json:"status"`
}

func generateText(e InEvent) string {
	return fmt.Sprintf("Window: %v, Middle: %v, Aisle: %v", e.Window, e.Middle, e.Aisle)
}

func sendNotification(server string, topic string, text string) error {
	m := "POST"
	u, err := url.Parse(server)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %v", err)
	}
	u = u.JoinPath(topic)

	b := strings.NewReader(text)
	req, err := http.NewRequest(m, u.String(), b)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Title", "Seatchecker")
	req.Header.Set("Tags", "airplane")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("invalid status code, expected: 200, received: %v", res.StatusCode)
	}

	return nil
}

func handler(e InEvent) (OutEvent, error) {
	log.Printf("Received Event: %v\n", e)

	topic := os.Getenv("SEATCHECKER_NTFY_TOPIC")
	if topic == "" {
		msg := "env var 'SEATCHECKER_NTFY_TOPIC' is not configured"
		fmt.Fprintln(os.Stderr, msg)
		return OutEvent{Status: 500}, errors.New(msg)
	}

	log.Println("Send notification.")
	server := "https://ntfy.sh"
	text := generateText(e)
	err := sendNotification(server, topic, text)
	if err != nil {
		msg := fmt.Sprintf("error: %v", err)
		fmt.Fprintln(os.Stderr, msg)
		return OutEvent{Status: 500}, errors.New(msg)
	}
	log.Println("Notification sent successfully.")

	return OutEvent{Status: 200}, nil
}

func main() {
	// lambda.Start(handler)
	resp, _ := handler(InEvent{Window: 4, Middle: 2, Aisle: 1})
	log.Println(resp)
}
