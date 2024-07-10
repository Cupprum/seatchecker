package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	lambdadetector "go.opentelemetry.io/contrib/detectors/aws/lambda"
	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var tracer trace.Tracer
var tp *sdktrace.TracerProvider

func setupOtel(ctx context.Context) (func(), error) {
	// Configure a new OTLP exporter using environment variables for sending data to Honeycomb over gRPC
	client := otlptracegrpc.NewClient()
	exp, err := otlptrace.New(ctx, client)
	if err != nil {
		return func() {}, fmt.Errorf("failed to initialize exporter: %e", err)
	}

	detector := lambdadetector.NewResourceDetector()
	res, err := detector.Detect(context.Background())
	if err != nil {
		fmt.Printf("failed to detect lambda resources: %v\n", err)
	}

	// Create a new tracer provider with a batch span processor and the otlp exporter
	tp = sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exp),
	)

	// Register the global Tracer provider
	otel.SetTracerProvider(tp)

	// Register the W3C trace context and baggage propagators so data is propagated across services/processes
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	tracer = tp.Tracer("seatchecher-notifier-lambda")

	// Handle shutdown to ensure all sub processes are closed correctly and telemetry is exported
	return func() {
		_ = exp.Shutdown(ctx)
		_ = tp.Shutdown(ctx)
	}, nil
}

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

func sendNotification(ctx context.Context, endpoint string, text string) error {
	_, span := tracer.Start(ctx, "send-notification")
	defer span.End()

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

	client := &http.Client{
		Transport: otelhttp.NewTransport(
			http.DefaultTransport,
			otelhttp.WithClientTrace(func(ctx context.Context) *httptrace.ClientTrace {
				return otelhttptrace.NewClientTrace(ctx)
			}),
		),
	}

	res, err := client.Do(req)
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
	defer func() { tp.ForceFlush(ctx) }()
	ctx, span := tracer.Start(ctx, "handler")
	defer span.End()

	throwErr := func(msg string) error {
		span.SetStatus(codes.Error, msg)
		fmt.Fprintln(os.Stderr, msg)
		err := errors.New(msg)
		span.RecordError(err)
		return err
	}

	log.Printf("Received Event: %v\n", e)

	ep := os.Getenv("SEATCHECKER_NTFY_ENDPOINT")
	if ep == "" {
		err := throwErr("env var 'SEATCHECKER_NTFY_ENDPOINT' is not configured")
		return OutEvent{Status: 500}, err
	}

	log.Println("Send notification.")
	text := generateText(e)

	span.SetAttributes(attribute.String("text", text))
	span.AddEvent("text generated")

	err := sendNotification(ctx, ep, text)
	if err != nil {
		msg := fmt.Sprintf("sendNotification failed, error: %v", err)
		err := throwErr(msg)
		return OutEvent{Status: 500}, err
	}
	log.Println("Notification sent successfully.")
	span.AddEvent("notification sent")

	return OutEvent{Status: 200}, nil
}

func main() {
	ctx := context.Background()

	cleanup, err := setupOtel(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer cleanup()

	lambda.Start(handler)
	// resp, _ := handler(ctx, InEvent{Window: 4, Middle: 2, Aisle: 1})
	// log.Println(resp)
}
