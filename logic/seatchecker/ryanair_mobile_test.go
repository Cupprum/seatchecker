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

func TestAccountLogin(t *testing.T) {
	em, pw := "john@doe.com", "password"
	e := RAuth{"customerid", "token"}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		rawB, _ := io.ReadAll(r.Body)
		b := map[string]any{}
		json.Unmarshal(rawB, &b)
		if em != b["email"] {
			t.Fatalf("wrong email, expected: %v, received: %v", e, b["email"])
		}
		if pw != b["password"] {
			t.Fatalf("wrong password, expected: %v, received: %v", e, b["password"])
		}

		// Create fake response
		res, _ := json.Marshal(e)
		fmt.Fprintln(w, string(res))
	}))
	defer ts.Close()

	// Check received response
	c := Client{scheme: "http", fqdn: ts.URL}
	rcvd, err := c.accountLogin(context.Background(), em, pw)
	if err != nil {
		t.Fatalf("failed to get account login: %v", err)
	}
	if e != rcvd {
		t.Fatalf("wrong response, expected: %v, received %v", e, rcvd)
	}
}
