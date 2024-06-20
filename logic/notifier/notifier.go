package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
)

// Event defines your lambda input and output data structure,
// and of course you can have different input and output data structure
type InEvent struct {
	Window int `json:"window"`
	Middle int `json:"middle"`
	Aisle  int `json:"aisle"`
}

type OutEvent struct {
	Status int `json:"status"`
}

func sendNotification(topic string, text string) error {
	m := "POST"
	// TODO: redo configuration of url while implementing testing. Where should i source config from?
	u, _ := url.Parse("https://ntfy.sh") // Errorhandling not required, not a variable.
	u = u.JoinPath(topic)

	b := strings.NewReader(text)
	req, _ := http.NewRequest(m, u.String(), b)
	req.Header.Set("Title", "Seatchecker")
	req.Header.Set("Tags", "airplane")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
		// TODO: implement exception handling
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		fmt.Fprintf(os.Stderr, "invalid status code, expected: 200, received: %v\n", res.StatusCode)
		os.Exit(1)
		// TODO: implement exception handling
	}

	return nil
}

func handler(request InEvent) (OutEvent, error) {
	log.Println("Program started.")

	log.Println(request)

	topic := os.Getenv("SEATCHECKER_NTFY_TOPIC")
	if topic == "" {
		fmt.Fprintf(os.Stderr, "env var 'SEATCHECKER_NTFY_TOPIC' is not configured")
		os.Exit(1)
	}

	log.Println("Send notification.")
	err := sendNotification(topic, "Window: 4, Middle: 2, Aisle: 0")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	log.Println("Program finished successfully.")
	return OutEvent{Status: 200}, nil
}

func main() {
	lambda.Start(handler)
	// resp, _ := handler(InEvent{Window: 4, Middle: 2, Aisle: 0})
	// log.Println(resp)
}
