package main

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
)

type EmptySeats struct {
	Window int `json:"window"`
	Middle int `json:"middle"`
	Aisle  int `json:"aisle"`
}

type Event struct {
	RyanairEmail    string     `json:"ryanair_email"`
	RyanairPassword string     `json:"ryanair_password"`
	NtfyTopic       string     `json:"ntfy_topic"`
	SeatState       EmptySeats `json:"seat_state"`
	Status          int        `json:"status"`
	Message         string     `json:"message"`
}

func (ss EmptySeats) generateText() string {
	return fmt.Sprintf("Window: %v, Middle: %v, Aisle: %v", ss.Window, ss.Middle, ss.Aisle)
}

func handler(ctx context.Context, e Event) (Event, error) {
	// TODO: configure opentelemetry
	defer func() { tp.ForceFlush(ctx) }()
	ctx, span := tr.Start(ctx, "handler")
	defer span.End()

	log.Printf("Received Event: %v\n", e)

	// Ryanair Mobile API.
	rmc := Client{scheme: "https", fqdn: "services-api.ryanair.com"}

	log.Printf("Start Ryanair account login for user: %s.\n", e.RyanairEmail)
	a, err := rmc.accountLogin(ctx, e.RyanairEmail, e.RyanairPassword)
	if err != nil {
		err = fmt.Errorf("login failed: %v", err)
		log.Printf("Error: %v\n", err)
		return Event{Status: 500, Message: err.Error()}, nil
	}
	log.Println("Account login finished successfully.")

	// Ryanair Browser API.
	rc := Client{scheme: "https", fqdn: "www.ryanair.com"}

	log.Println("Query Ryanair for seats.")
	es, err := rc.getEmptySeats(ctx, a)
	if err != nil {
		err = fmt.Errorf("failed to query ryanair for seats, error: %v", err)
		log.Printf("Error: %v\n", err)
		return Event{Status: 500, Message: err.Error()}, nil
	}
	log.Println("Seats from Ryanair retrieved successfully.")

	pTxt := e.SeatState.generateText()
	log.Printf("Previous execution: %v", pTxt)

	cTxt := es.generateText()
	log.Printf("Current execution: %v", cTxt)

	if pTxt != cTxt {
		nc := Client{scheme: "https", fqdn: "ntfy.sh"}

		log.Println("Send notification.")
		err := nc.sendNotification(ctx, e.NtfyTopic, cTxt)
		if err != nil {
			err = fmt.Errorf("failed to send notification, error: %v", err)
			log.Printf("Error: %v\n", err)
			return Event{Status: 500, Message: err.Error()}, nil
		}
		log.Println("Notification sent successfully.")
	}

	e.SeatState = es
	e.Status = 200

	log.Println("Program finished successfully.")
	return e, nil
}

func main() {
	ctx := context.Background()

	// TODO: try to simplify OTEL through ADOT.
	defer setupOtel(ctx)()

	lambda.Start(handler)
	// i := Event{
	// 	RyanairEmail:    os.Getenv("SEATCHECKER_RYANAIR_EMAIL"),
	// 	RyanairPassword: os.Getenv("SEATCHECKER_RYANAIR_PASSWORD"),
	// 	NtfyTopic:       os.Getenv("SEATCHECKER_NTFY_TOPIC"),
	// 	SeatState: EmptySeats{
	// 		Window: 99,
	// 		Middle: 99,
	// 		Aisle:  99,
	// 	},
	// }
	// resp, _ := handler(ctx, i)
	// log.Println(resp)
}
