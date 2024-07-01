package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"
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

func sendNotification(endpoint string, text string) error {
	m := "POST"
	u, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %v", err)
	}

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

func handler(ctx context.Context, e InEvent) (OutEvent, error) {
	log.Printf("Received Event: %v\n", e)

	ep := os.Getenv("SEATCHECKER_NTFY_ENDPOINT")
	if ep == "" {
		msg := "env var 'SEATCHECKER_NTFY_ENDPOINT' is not configured"
		fmt.Fprintln(os.Stderr, msg)
		return OutEvent{Status: 500}, errors.New(msg)
	}

	log.Println("Send notification.")
	text := generateText(e)
	err := sendNotification(ep, text)
	if err != nil {
		msg := fmt.Sprintf("error: %v", err)
		fmt.Fprintln(os.Stderr, msg)
		return OutEvent{Status: 500}, errors.New(msg)
	}
	log.Println("Notification sent successfully.")

	return OutEvent{Status: 200}, nil
}

func main() {
	lambda.Start(otellambda.InstrumentHandler(handler))
	// resp, _ := handler(context.Background(), InEvent{Window: 4, Middle: 2, Aisle: 1})
	// log.Println(resp)
}
