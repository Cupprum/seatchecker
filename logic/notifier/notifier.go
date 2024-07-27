package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
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

func executePostRequest(ctx context.Context, endpoint string, headers map[string]string, body io.Reader) error {
	m := "POST"
	u, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, m, u.String(), body)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

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

func sendNotification(ctx context.Context, endpoint string, text string) error {
	ctx, span := tracer.Start(ctx, "send notification")
	defer span.End()

	h := map[string]string{
		"Title": "Seatchecker",
		"Tags":  "airplane",
	}
	b := strings.NewReader(text)

	err := executePostRequest(ctx, endpoint, h, b)
	if err != nil {
		return fmt.Errorf("failed to send notification: %v", err)
	}

	return nil
}

func handler(ctx context.Context, e InEvent) (OutEvent, error) {
	defer func() { tp.ForceFlush(ctx) }()
	ctx, span := tracer.Start(ctx, "handler")
	defer span.End()

	log.Printf("Received Event: %v\n", e)

	throwErr := func(msg string) error {
		span.SetStatus(codes.Error, msg)
		fmt.Fprintln(os.Stderr, msg)
		err := errors.New(msg)
		span.RecordError(err, trace.WithStackTrace(true))
		return err
	}

	ep := os.Getenv("SEATCHECKER_NTFY_ENDPOINT")
	if ep == "" {
		err := throwErr("env var 'SEATCHECKER_NTFY_ENDPOINT' is not configured")
		return OutEvent{Status: 500}, err
	}

	log.Println("Send notification.")

	text := generateText(e)
	span.AddEvent("text generated")
	span.SetAttributes(attribute.String("text", text))

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
