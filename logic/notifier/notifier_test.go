package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestGenerateText(t *testing.T) {
	o := generateText(InEvent{Window: 4, Middle: 0, Aisle: 2})
	ex := "Window: 4, Middle: 0, Aisle: 2"
	if ex != o {
		t.Fatalf("wrong output, expected: %v, received: %v", ex, o)
	}
}

func TestExecutePostRequest(t *testing.T) {
	os.Setenv("OTEL_SERVICE_NAME", "seatchecker-notifier-lambda-test")

	ctx := context.Background()
	cleanup, err := setupOtel(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("wrong http method, expected: POST, received: %v", r.Method)
		}
		if r.Header["Testkey"][0] != "Testval" {
			t.Fatalf("wrong title, expected: Seatchecker, received: %v", r.Header["Title"][0])
		}
	}))
	defer ts.Close()

	h := map[string]string{
		"Testkey": "Testval",
	}
	b := strings.NewReader("test-text")
	err = executePostRequest(ctx, ts.URL, h, b)

	if err != nil {
		t.Fatalf("error: %v", err)
	}
}

func TestSendNotification(t *testing.T) {
	os.Setenv("OTEL_SERVICE_NAME", "seatchecker-notifier-lambda-test")

	ctx := context.Background()
	cleanup, err := setupOtel(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("wrong http method, expected: POST, received: %v", r.Method)
		}
		if r.Header["Title"][0] != "Seatchecker" {
			t.Fatalf("wrong title, expected: Seatchecker, received: %v", r.Header["Title"][0])
		}
		if r.Header["Tags"][0] != "airplane" {
			t.Fatalf("wrong tags, expected: airplane, received: %v", r.Header["Tags"][0])
		}
	}))
	defer ts.Close()

	err = sendNotification(ctx, ts.URL, "test-text")

	if err != nil {
		t.Fatalf("error: %v", err)
	}
}

func TestHandler(t *testing.T) {
	os.Setenv("OTEL_SERVICE_NAME", "seatchecker-notifier-lambda-test")

	ctx := context.Background()
	cleanup, err := setupOtel(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer ts.Close()

	os.Setenv("SEATCHECKER_NTFY_ENDPOINT", ts.URL)

	i := InEvent{Window: 4, Middle: 0, Aisle: 2}
	o, err := handler(ctx, i)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if o.Status != 200 {
		t.Fatalf("invalid response code, expected: 200, received: %v", o.Status)
	}
}
