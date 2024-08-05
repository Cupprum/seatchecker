package main

import (
	"context"
	"fmt"
	"log"
	"os"
)

type Event struct {
	RyanairEmail    string `json:"ryanair_email"`
	RyanairPassword string `json:"ryanair_password"`
	NtfyTopic       string `json:"ntfy_topic"`
	Window          int    `json:"window"`
	Middle          int    `json:"middle"`
	Aisle           int    `json:"aisle"`
	Status          int    `json:"status"`
	Message         string `json:"message"`
}

func generateText(w int, m int, a int) string {
	return fmt.Sprintf("Window: %v, Middle: %v, Aisle: %v", w, m, a)
}

func handler(ctx context.Context, e Event) (Event, error) {
	// TODO: configure opentelemetry
	defer func() { tp.ForceFlush(ctx) }()
	ctx, span := tr.Start(ctx, "handler")
	defer span.End()

	log.Printf("Received Event: %v\n", e)

	// TODO: verify the input event

	log.Println("Query Ryanair for seats.")
	w, m, a, err := queryRyanair(e.RyanairEmail, e.RyanairPassword)
	if err != nil {
		err = fmt.Errorf("failed to query ryanair for seats, error: %v", err)
		log.Fatalf("Error: %v\n", err)
		return Event{Status: 500, Message: err.Error()}, err
	}
	log.Println("Seats from Ryanair retrieved successfully.")

	pTxt := generateText(e.Window, e.Middle, e.Aisle)
	cTxt := generateText(w, m, a)
	log.Printf("Previous execution: %v", pTxt)
	log.Printf("Current execution: %v", cTxt)

	if pTxt != cTxt {
		log.Println("Send notification.")
		err := sendNotification(context.Background(), e.NtfyTopic, cTxt)
		if err != nil {
			err = fmt.Errorf("failed to send notification, error: %v", err)
			log.Fatalf("Error: %v\n", err)
			return Event{Status: 500, Message: err.Error()}, err
		}
		log.Println("Notification sent successfully.")
	}

	// TODO: polish updating event and generating the message body
	e.Window = w
	e.Middle = m
	e.Aisle = a
	e.Status = 200

	log.Println("Program finished successfully.")
	return e, nil
}

func main() {
	ctx := context.Background()

	// TODO: try to simplify OTEL through ADOT.
	cleanup, err := setupOtel(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer cleanup()

	// lambda.Start(handler)
	i := Event{
		RyanairEmail:    os.Getenv("SEATCHECKER_RYANAIR_EMAIL"),
		RyanairPassword: os.Getenv("SEATCHECKER_RYANAIR_PASSWORD"),
		NtfyTopic:       os.Getenv("SEATCHECKER_NTFY_TOPIC"),
		Window:          99,
		Middle:          99,
		Aisle:           99,
	}
	resp, _ := handler(ctx, i)
	log.Println(resp)

}
