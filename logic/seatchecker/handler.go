package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Event struct {
	RyanairEmail    string     `json:"ryanair_email"`
	RyanairPassword string     `json:"ryanair_password"`
	NtfyTopic       string     `json:"ntfy_topic"`
	SeatState       EmptySeats `json:"seat_state"`
	Status          int        `json:"status"`
	Message         string     `json:"message"`
	Departure       string     `json:"departure"`
}

type EmptySeats struct {
	Window int `json:"window"`
	Middle int `json:"middle"`
	Aisle  int `json:"aisle"`
}

// Wrapped in a function for testability purpose.
func (es EmptySeats) generateText() string {
	return fmt.Sprintf("Window: %v, Middle: %v, Aisle: %v", es.Window, es.Middle, es.Aisle)
}

// TODO: test this part of logic.
func nextDeparture(js []string) (string, error) {
	n := time.Now().UTC()

	// Find the upcoming flight.
	var pt time.Time
	var err error
	for _, j := range js {
		// RFC3339 is the formet Ryanair uses for time.
		pt, err = time.Parse(time.RFC3339, j)
		if err != nil {
			return "", fmt.Errorf("error parsing time: %v", err)
		}
		if pt.Before(n) {
			continue
		} else {
			break
		}
	}

	return pt.UTC().String(), nil
}

func handler(ctx context.Context, e Event) (Event, error) {
	log.Println("Started Lambda execution.")

	defer func() { tp.ForceFlush(ctx) }()
	ctx, span := tr.Start(ctx, "handler")
	defer span.End()
	span.SetAttributes(
		attribute.Int("window", e.SeatState.Window),
		attribute.Int("middle", e.SeatState.Middle),
		attribute.Int("aisle", e.SeatState.Aisle))

	// Helper function to throw error.
	throwErr := func(err error) (Event, error) {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, err.Error())
		log.Printf("Error: %v\n", err)
		// Returning nil error, as lambda finished.
		// The error which happend in logic is returned through Message of Event response.
		return Event{Status: 500, Message: err.Error()}, nil
	}

	// Ryanair Mobile API.
	rmc := Client{scheme: "https", fqdn: "services-api.ryanair.com"}

	log.Printf("Start Ryanair account login for user: %s.\n", e.RyanairEmail)
	a, err := rmc.accountLogin(ctx, e.RyanairEmail, e.RyanairPassword)
	if err != nil {
		err := fmt.Errorf("login failed: %v", err)
		return throwErr(err)
	}
	span.AddEvent("Account login finished successfully.")

	// Ryanair Browser API.
	rc := Client{scheme: "https", fqdn: "www.ryanair.com"}

	log.Println("Query Ryanair for seats.")
	es, js, err := rc.getEmptySeats(ctx, a)
	if err != nil {
		err := fmt.Errorf("failed to query ryanair for seats, error: %v", err)
		return throwErr(err)
	}
	span.AddEvent("Seats from Ryanair retrieved successfully.")

	pTxt := e.SeatState.generateText()
	log.Printf("Previous execution: %v", pTxt)
	span.AddEvent("Previous execution text generated.", trace.WithAttributes(
		attribute.String("previous_execution", pTxt)))

	cTxt := es.generateText()
	log.Printf("Current execution: %v", cTxt)
	span.AddEvent("Current execution text generated.", trace.WithAttributes(
		attribute.String("current_execution", cTxt)))

	if pTxt != cTxt {
		// Send notification that there is a change in seat availability.
		nc := Client{scheme: "https", fqdn: "ntfy.sh"}

		log.Println("Send notification.")
		err := nc.sendNotification(ctx, e.NtfyTopic, cTxt)
		if err != nil {
			err = fmt.Errorf("failed to send notification, error: %v", err)
			return throwErr(err)
		}
		span.AddEvent("Notification sent successfully.")
	}

	// Execute on first run.
	if e.Departure == "" {
		e.Departure, err = nextDeparture(js)
		if err != nil {
			err = fmt.Errorf("error calculating next departure: %v", err)
			return throwErr(err)
		}
	}

	// Execute on last run.
	d, err := time.Parse(time.RFC3339, e.Departure)
	if err != nil {
		err = fmt.Errorf("error parsing time: %v", err)
		return throwErr(err)
	}
	if time.Now().UTC().After(d) {
		es = EmptySeats{0, 0, 0}
	}

	e.SeatState = es
	e.Status = 200

	span.AddEvent("Program finished successfully.")
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
