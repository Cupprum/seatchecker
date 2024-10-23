package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendNotification(t *testing.T) {
	ctx := context.Background()
	defer setupOtel(ctx)()

	eTopic := "test_topic"
	eMessage := "test_text"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.Method != "POST" {
			t.Fatalf("wrong http method, expected: POST, received: %v", r.Method)
		}
		rawB, _ := io.ReadAll(r.Body)
		b := map[string]any{}
		json.Unmarshal(rawB, &b)
		if b["topic"] != eTopic {
			t.Fatalf("wrong topic name, expected: %v, received: %v", eTopic, b["topic"])
		}
		if b["message"] != eMessage {
			t.Fatalf("wrong message, expected: %v, received: %v", eMessage, b["message"])
		}

		// Create fake response
		fmt.Fprintln(w, "{}")
	}))
	defer ts.Close()

	// Check received response
	c := Client{scheme: "http", fqdn: ts.URL}
	err := c.sendNotification(ctx, eTopic, eMessage)
	if err != nil {
		t.Fatalf("failed to send notification: %v", err)
	}
}
