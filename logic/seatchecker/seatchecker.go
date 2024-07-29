package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
)

type InEvent struct {
	RyanairEmail    string `json:"ryanair_email"`
	RyanairPassword string `json:"ryanair_password"`
	NtfyTopic       string `json:"ntfy_topic"`
	Window          int    `json:"window"`
	Middle          int    `json:"middle"`
	Aisle           int    `json:"aisle"`
}

type OutEvent struct {
	Status int `json:"status"`
}

type Client struct {
	scheme string
	fqdn   string
}

type Request struct {
	method      string
	scheme      string
	fqdn        string
	path        string
	queryParams url.Values
	headers     http.Header
	body        any
}

func (r Request) creator() (*http.Request, error) {
	u, err := url.Parse(r.fqdn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %v", err)
	}
	u = u.JoinPath(r.path)              // Specify path.
	u.Scheme = r.scheme                 // Specify scheme.
	u.RawQuery = r.queryParams.Encode() // Specify query string parameters.

	buf := []byte{} // If payload not specified, send empty buffer.
	if r.body != nil {
		buf, err = json.Marshal(r.body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %v", err)
		}
	}

	// TODO: add context to the request creation.
	req, err := http.NewRequest(r.method, u.String(), bytes.NewBuffer(buf))
	if err != nil {
		return nil, fmt.Errorf("failed to form request: %v", err)
	}

	if r.headers != nil {
		req.Header = r.headers
	}
	req.Header.Add("Content-Type", "application/json") // Add default headers

	return req, nil
}

func httpsRequest[T any](req Request) (T, error) {
	var nilT T // Empty response for errors.

	r, err := req.creator()
	if err != nil {
		return nilT, fmt.Errorf("failed to create request: %v", err)
	}

	c := &http.Client{}
	res, err := c.Do(r)
	if err != nil {
		return nilT, fmt.Errorf("failed to execute request: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nilT, fmt.Errorf("request return invalid code: %v", res.StatusCode)
	}

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nilT, fmt.Errorf("failed to read response: %v", err)
	}

	var t T
	if err := json.Unmarshal(b, &t); err != nil {
		return nilT, fmt.Errorf("failed to unmarshal Json response: %v", err)
	}

	return t, nil
}

func httpsRequestGet[T any](c Client, path string, queryParams url.Values, headers http.Header) (T, error) {
	r := Request{
		"GET",
		c.scheme,
		c.fqdn,
		path,
		queryParams,
		headers,
		nil,
	}
	return httpsRequest[T](r)
}

func httpsRequestPost[T any](c Client, path string, body any) (T, error) {
	r := Request{
		"POST",
		c.scheme,
		c.fqdn,
		path,
		nil,
		nil,
		body,
	}
	return httpsRequest[T](r)
}

func generateText(w int, m int, a int) string {
	return fmt.Sprintf("Window: %v, Middle: %v, Aisle: %v", w, m, a)
}

func handler(ctx context.Context, e InEvent) (OutEvent, error) {
	// TODO: configure opentelemetry
	defer func() { tp.ForceFlush(ctx) }()
	ctx, span := tr.Start(ctx, "handler")
	defer span.End()

	log.Printf("Received Event: %v\n", e)

	// TODO: verify the input event

	w, m, a, err := queryRyanair(e.RyanairEmail, e.RyanairPassword)
	if err != nil {
		log.Fatal("Failed to query Ryanair for seats.")
		return OutEvent{Status: 500}, err
	}
	log.Println("Seats from Ryanair retrieved successfully.")

	pTxt := generateText(e.Window, e.Middle, e.Aisle)
	cTxt := generateText(w, m, a)
	log.Printf("Previous execution: %v", pTxt)
	log.Printf("Current execution: %v", cTxt)

	if pTxt != cTxt {
		err := sendNotification(context.Background(), e.NtfyTopic, cTxt)
		if err != nil {
			log.Fatal("Failed to send notification.")
			return OutEvent{Status: 500}, err
		}
		log.Println("Notification sent successfully.")
	}

	// TODO: add event to SQS

	log.Println("Program finished successfully.")
	return OutEvent{
		Status: 200,
	}, nil
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
	i := InEvent{
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
