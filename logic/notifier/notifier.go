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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/trace"
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
	ctx := context.Background()

	// Configure a new OTLP exporter using environment variables for sending data to Honeycomb over gRPC
	client := otlptracegrpc.NewClient()
	exp, err := otlptrace.New(ctx, client)
	if err != nil {
		log.Fatalf("failed to initialize exporter: %e", err)
	}

	// Create a new tracer provider with a batch span processor and the otlp exporter
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exp),
	)

	// Handle shutdown to ensure all sub processes are closed correctly and telemetry is exported
	defer func() {
		_ = exp.Shutdown(ctx)
		_ = tp.Shutdown(ctx)
	}()

	// Register the global Tracer provider
	otel.SetTracerProvider(tp)

	// Register the W3C trace context and baggage propagators so data is propagated across services/processes
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	lambda.Start(otellambda.InstrumentHandler(handler))
	// resp, _ := handler(context.Background(), InEvent{Window: 4, Middle: 2, Aisle: 1})
	// log.Println(resp)
}
