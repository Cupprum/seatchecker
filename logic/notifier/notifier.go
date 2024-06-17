package main

import (
	"fmt"
	"log"
	"net/http"
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

func handler(request InEvent) (OutEvent, error) {
	log.Println("Program started.")

	log.Println(request)

	topic := os.Getenv("SEATCHECKER_NTFY_TOPIC")
	if topic == "" {
		fmt.Fprintf(os.Stderr, "env var 'SEATCHECKER_NTFY_TOPIC' is not configured")
		os.Exit(1)
	}

	resp, err := http.Post(fmt.Sprintf("https://ntfy.sh/%v", topic), "text/plain", strings.NewReader("Test1234"))
	if resp.Status != "200" || err != nil {
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
